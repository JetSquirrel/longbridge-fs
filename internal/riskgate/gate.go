package riskgate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"longbridge-fs/internal/model"
)

// Gate is the pre-trade risk control gate
type Gate struct {
	root   string
	policy model.RiskPolicy
	rules  model.PreTradeRules
	limits model.PositionLimits
}

// NewGate creates a new risk gate instance
func NewGate(root string) (*Gate, error) {
	g := &Gate{root: root}

	// Load policy
	policyPath := filepath.Join(root, "trade", "risk", "policy.json")
	policyData, err := os.ReadFile(policyPath)
	if err != nil {
		// If policy file doesn't exist, risk gate is disabled
		g.policy.Enabled = false
		return g, nil
	}
	if err := json.Unmarshal(policyData, &g.policy); err != nil {
		return nil, fmt.Errorf("failed to parse policy.json: %w", err)
	}

	// If policy is disabled, return early
	if !g.policy.Enabled {
		return g, nil
	}

	// Load pre-trade rules
	rulesPath := filepath.Join(root, "trade", "risk", "pre_trade.json")
	rulesData, err := os.ReadFile(rulesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read pre_trade.json: %w", err)
	}
	if err := json.Unmarshal(rulesData, &g.rules); err != nil {
		return nil, fmt.Errorf("failed to parse pre_trade.json: %w", err)
	}

	// Load position limits
	limitsPath := filepath.Join(root, "trade", "risk", "position_limits.json")
	limitsData, err := os.ReadFile(limitsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read position_limits.json: %w", err)
	}
	if err := json.Unmarshal(limitsData, &g.limits); err != nil {
		return nil, fmt.Errorf("failed to parse position_limits.json: %w", err)
	}

	return g, nil
}

// CheckOrder performs pre-trade validation on an order
func (g *Gate) CheckOrder(order *model.ParsedOrder, accountState *model.AccountState) model.RiskCheckResult {
	// If policy is disabled, pass all orders
	if !g.policy.Enabled || !g.policy.PreTradeChecks {
		return model.RiskCheckResult{Passed: true}
	}

	// Check if trading is halted
	dailyLimits, err := g.loadDailyLimits()
	if err == nil && dailyLimits.IsHalted {
		return model.RiskCheckResult{
			Passed: false,
			Rule:   "trading_halted",
			Reason: fmt.Sprintf("Trading is halted: %s", *dailyLimits.HaltReason),
		}
	}

	// Check blocked symbols
	if result := g.checkBlockedSymbols(order); !result.Passed {
		return result
	}

	// Check allowed symbols (if whitelist is configured)
	if result := g.checkAllowedSymbols(order); !result.Passed {
		return result
	}

	// Check allowed sides
	if result := g.checkAllowedSides(order); !result.Passed {
		return result
	}

	// Check order size limits
	if result := g.checkOrderSize(order, accountState); !result.Passed {
		return result
	}

	// Check position limits
	if result := g.checkPositionLimits(order, accountState); !result.Passed {
		return result
	}

	// Check order frequency
	if result := g.checkOrderFrequency(); !result.Passed {
		return result
	}

	return model.RiskCheckResult{Passed: true}
}

// checkBlockedSymbols checks if the symbol is blocked
func (g *Gate) checkBlockedSymbols(order *model.ParsedOrder) model.RiskCheckResult {
	for _, blocked := range g.rules.BlockedSymbols {
		if order.Symbol == blocked {
			return model.RiskCheckResult{
				Passed: false,
				Rule:   "blocked_symbol",
				Reason: fmt.Sprintf("Symbol %s is blocked by risk policy", order.Symbol),
			}
		}
	}
	return model.RiskCheckResult{Passed: true}
}

// checkAllowedSymbols checks if the symbol is in the allowed list (if configured)
func (g *Gate) checkAllowedSymbols(order *model.ParsedOrder) model.RiskCheckResult {
	// If no allowed symbols configured, pass
	if len(g.rules.AllowedSymbols) == 0 {
		return model.RiskCheckResult{Passed: true}
	}

	for _, allowed := range g.rules.AllowedSymbols {
		if order.Symbol == allowed {
			return model.RiskCheckResult{Passed: true}
		}
	}

	return model.RiskCheckResult{
		Passed: false,
		Rule:   "symbol_not_allowed",
		Reason: fmt.Sprintf("Symbol %s is not in the allowed list", order.Symbol),
	}
}

// checkAllowedSides checks if the order side is allowed
func (g *Gate) checkAllowedSides(order *model.ParsedOrder) model.RiskCheckResult {
	if len(g.rules.AllowedSides) == 0 {
		return model.RiskCheckResult{Passed: true}
	}

	for _, allowed := range g.rules.AllowedSides {
		if order.Side == allowed {
			return model.RiskCheckResult{Passed: true}
		}
	}

	return model.RiskCheckResult{
		Passed: false,
		Rule:   "side_not_allowed",
		Reason: fmt.Sprintf("Order side %s is not allowed by risk policy", order.Side),
	}
}

// checkOrderSize checks if the order size exceeds limits
func (g *Gate) checkOrderSize(order *model.ParsedOrder, accountState *model.AccountState) model.RiskCheckResult {
	// Calculate total equity
	totalEquity := g.calculateTotalEquity(accountState)
	if totalEquity <= 0 {
		// Cannot validate without equity info
		return model.RiskCheckResult{Passed: true}
	}

	// Parse order quantity
	qty, err := strconv.ParseFloat(order.Qty, 64)
	if err != nil {
		return model.RiskCheckResult{
			Passed: false,
			Rule:   "invalid_quantity",
			Reason: fmt.Sprintf("Invalid order quantity: %s", order.Qty),
		}
	}

	// Parse order price (approximate order value)
	price := 0.0
	if order.Price != "" {
		price, _ = strconv.ParseFloat(order.Price, 64)
	}

	// Estimate order value
	orderValue := qty * price

	// Check max single order value
	if g.rules.MaxSingleOrderValue > 0 && orderValue > g.rules.MaxSingleOrderValue {
		return model.RiskCheckResult{
			Passed: false,
			Rule:   "max_single_order_value",
			Reason: fmt.Sprintf("Order value %.2f exceeds limit %.2f", orderValue, g.rules.MaxSingleOrderValue),
		}
	}

	// Check max single order percentage
	if g.rules.MaxSingleOrderPct > 0 {
		orderPct := orderValue / totalEquity
		if orderPct > g.rules.MaxSingleOrderPct {
			return model.RiskCheckResult{
				Passed: false,
				Rule:   "max_single_order_pct",
				Reason: fmt.Sprintf("Order value %.2f = %.1f%% of equity, limit %.1f%%",
					orderValue, orderPct*100, g.rules.MaxSingleOrderPct*100),
			}
		}
	}

	return model.RiskCheckResult{Passed: true}
}

// checkPositionLimits checks if the order would violate position limits
func (g *Gate) checkPositionLimits(order *model.ParsedOrder, accountState *model.AccountState) model.RiskCheckResult {
	// For BUY orders, check if we're already at max positions count
	if order.Side == "BUY" {
		currentPositionCount := len(accountState.Positions)

		// Check if symbol already in positions
		hasPosition := false
		for _, pos := range accountState.Positions {
			if pos.Symbol == order.Symbol {
				hasPosition = true
				break
			}
		}

		// If it's a new position, check max count
		if !hasPosition && g.limits.MaxPositionsCount > 0 {
			if currentPositionCount >= g.limits.MaxPositionsCount {
				return model.RiskCheckResult{
					Passed: false,
					Rule:   "max_positions_count",
					Reason: fmt.Sprintf("Already at max positions limit (%d)", g.limits.MaxPositionsCount),
				}
			}
		}
	}

	// Check per-symbol limits
	if symbolLimit, ok := g.limits.PerSymbolLimits[order.Symbol]; ok {
		totalEquity := g.calculateTotalEquity(accountState)
		if totalEquity > 0 {
			// Calculate what the position would be after this order
			qty, _ := strconv.ParseFloat(order.Qty, 64)
			price, _ := strconv.ParseFloat(order.Price, 64)
			orderValue := qty * price

			// For simplicity, just check if single order exceeds symbol limit
			if symbolLimit.MaxPct > 0 {
				orderPct := orderValue / totalEquity
				if orderPct > symbolLimit.MaxPct {
					return model.RiskCheckResult{
						Passed: false,
						Rule:   "per_symbol_limit",
						Reason: fmt.Sprintf("Order would exceed per-symbol limit for %s (%.1f%% > %.1f%%)",
							order.Symbol, orderPct*100, symbolLimit.MaxPct*100),
					}
				}
			}
		}
	}

	return model.RiskCheckResult{Passed: true}
}

// checkOrderFrequency checks if order frequency limits are exceeded
func (g *Gate) checkOrderFrequency() model.RiskCheckResult {
	if !g.policy.OrderFrequency.Enabled {
		return model.RiskCheckResult{Passed: true}
	}

	dailyLimits, err := g.loadDailyLimits()
	if err != nil {
		// Cannot check without daily limits
		return model.RiskCheckResult{Passed: true}
	}

	// Check hourly limit
	if g.policy.OrderFrequency.MaxOrdersPerHour > 0 {
		if dailyLimits.OrdersThisHour >= g.policy.OrderFrequency.MaxOrdersPerHour {
			return model.RiskCheckResult{
				Passed: false,
				Rule:   "max_orders_per_hour",
				Reason: fmt.Sprintf("Hourly order limit reached (%d)", g.policy.OrderFrequency.MaxOrdersPerHour),
			}
		}
	}

	// Check daily limit
	if g.policy.OrderFrequency.MaxOrdersPerDay > 0 {
		if dailyLimits.OrdersToday >= g.policy.OrderFrequency.MaxOrdersPerDay {
			return model.RiskCheckResult{
				Passed: false,
				Rule:   "max_orders_per_day",
				Reason: fmt.Sprintf("Daily order limit reached (%d)", g.policy.OrderFrequency.MaxOrdersPerDay),
			}
		}
	}

	return model.RiskCheckResult{Passed: true}
}

// RecordViolation records a risk rule violation
func (g *Gate) RecordViolation(order *model.ParsedOrder, result model.RiskCheckResult) error {
	violation := model.RiskViolation{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Rule:      result.Rule,
		IntentID:  order.IntentID,
		Detail:    result.Reason,
		Action:    "REJECTED",
	}

	violationsPath := filepath.Join(g.root, "trade", "risk", "violations.jsonl")
	f, err := os.OpenFile(violationsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open violations.jsonl: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(violation)
	if err != nil {
		return fmt.Errorf("failed to marshal violation: %w", err)
	}

	if _, err := f.WriteString(string(data) + "\n"); err != nil {
		return fmt.Errorf("failed to write violation: %w", err)
	}

	return nil
}

// UpdateStatus updates the risk gate status
func (g *Gate) UpdateStatus(passed bool) error {
	statusPath := filepath.Join(g.root, "trade", "risk", "status.json")

	var status model.RiskStatus
	data, err := os.ReadFile(statusPath)
	if err == nil {
		_ = json.Unmarshal(data, &status)
	}

	status.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	status.ChecksToday++
	if passed {
		status.ChecksPassed++
	} else {
		status.ChecksRejected++
	}

	data, err = json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal status: %w", err)
	}

	if err := os.WriteFile(statusPath, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("failed to write status: %w", err)
	}

	return nil
}

// IncrementOrderCount increments the order count in daily limits
func (g *Gate) IncrementOrderCount() error {
	dailyLimits, err := g.loadDailyLimits()
	if err != nil {
		// Initialize if doesn't exist
		dailyLimits = &model.DailyLimits{
			Date: time.Now().UTC().Format("2006-01-02"),
		}
	}

	// Check if it's a new day
	today := time.Now().UTC().Format("2006-01-02")
	if dailyLimits.Date != today {
		// Reset counters for new day
		dailyLimits.Date = today
		dailyLimits.OrdersThisHour = 0
		dailyLimits.OrdersToday = 0
	}

	dailyLimits.OrdersThisHour++
	dailyLimits.OrdersToday++

	return g.saveDailyLimits(dailyLimits)
}

// Helper functions

func (g *Gate) calculateTotalEquity(accountState *model.AccountState) float64 {
	total := 0.0

	// Sum up all cash
	for _, cash := range accountState.Cash {
		total += cash.Available + cash.Frozen + cash.Settling
	}

	// Add position values (need current prices, simplified here)
	for _, pos := range accountState.Positions {
		qty, _ := strconv.ParseFloat(pos.Quantity, 64)
		total += qty * pos.CostPrice // Using cost price as approximation
	}

	return total
}

func (g *Gate) loadDailyLimits() (*model.DailyLimits, error) {
	limitsPath := filepath.Join(g.root, "trade", "risk", "daily_limits.json")
	data, err := os.ReadFile(limitsPath)
	if err != nil {
		return nil, err
	}

	var limits model.DailyLimits
	if err := json.Unmarshal(data, &limits); err != nil {
		return nil, err
	}

	return &limits, nil
}

func (g *Gate) saveDailyLimits(limits *model.DailyLimits) error {
	limitsPath := filepath.Join(g.root, "trade", "risk", "daily_limits.json")

	data, err := json.MarshalIndent(limits, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal daily limits: %w", err)
	}

	if err := os.WriteFile(limitsPath, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("failed to write daily limits: %w", err)
	}

	return nil
}

// ShouldWarnOnly returns true if the policy is in WARN mode
func (g *Gate) ShouldWarnOnly() bool {
	return g.policy.Enabled && strings.ToUpper(g.policy.Mode) == "WARN"
}

// IsEnabled returns true if the risk gate is enabled
func (g *Gate) IsEnabled() bool {
	return g.policy.Enabled
}
