package portfolio

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"longbridge-fs/internal/model"
)

// SyncCurrent reads account/state.json and quote/hold/*/overview.json
// to generate portfolio/current.json with position weights and market values.
func SyncCurrent(root string) error {
	// Read account state
	stateData, err := os.ReadFile(filepath.Join(root, "account", "state.json"))
	if err != nil {
		return fmt.Errorf("read account state: %w", err)
	}
	var state model.AccountState
	if err := json.Unmarshal(stateData, &state); err != nil {
		return fmt.Errorf("parse account state: %w", err)
	}

	// Calculate total equity
	var totalCash float64
	for _, c := range state.Cash {
		totalCash += c.Available
	}

	current := model.CurrentPortfolio{
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Positions: make(map[string]model.CurrentPosition),
		Cash:      totalCash,
	}

	var totalPositionValue float64

	// Process each position
	for _, pos := range state.Positions {
		qty := parseFloat(pos.Quantity)
		if qty == 0 {
			continue
		}

		// Look up current price
		overviewPath := filepath.Join(root, "quote", "hold", pos.Symbol, "overview.json")
		overviewData, err := os.ReadFile(overviewPath)
		if err != nil {
			// Skip if no quote available
			continue
		}

		var overview model.QuoteOverview
		if err := json.Unmarshal(overviewData, &overview); err != nil {
			continue
		}

		marketValue := qty * overview.Last
		totalPositionValue += marketValue

		current.Positions[pos.Symbol] = model.CurrentPosition{
			Qty:         qty,
			MarketValue: marketValue,
			Weight:      0, // will be calculated after total equity is known
			AvgCost:     pos.CostPrice,
		}
	}

	current.TotalEquity = totalCash + totalPositionValue
	current.CashPct = 0
	if current.TotalEquity > 0 {
		current.CashPct = totalCash / current.TotalEquity
	}

	// Calculate weights
	for symbol, pos := range current.Positions {
		pos.Weight = 0
		if current.TotalEquity > 0 {
			pos.Weight = pos.MarketValue / current.TotalEquity
		}
		current.Positions[symbol] = pos
	}

	// Write portfolio/current.json
	data, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal current portfolio: %w", err)
	}

	portfolioDir := filepath.Join(root, "portfolio")
	if err := os.MkdirAll(portfolioDir, 0755); err != nil {
		return fmt.Errorf("create portfolio dir: %w", err)
	}

	return os.WriteFile(filepath.Join(portfolioDir, "current.json"), append(data, '\n'), 0644)
}

// parseFloat converts string quantity to float64
func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
