package portfolio

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"longbridge-fs/internal/market"
	"longbridge-fs/internal/model"
)

// SyncCurrent reads account/state.json and quote/hold/*/overview.json,
// computes the live portfolio snapshot, and writes portfolio/current.json.
func SyncCurrent(root string) error {
	// Read account state
	stateData, err := os.ReadFile(filepath.Join(root, "account", "state.json"))
	if err != nil {
		return fmt.Errorf("failed to read account state: %w", err)
	}
	var state model.AccountState
	if err := json.Unmarshal(stateData, &state); err != nil {
		return fmt.Errorf("failed to parse account state: %w", err)
	}

	positions := make(map[string]model.CurrentPosition)
	totalMktValue := 0.0

	for _, p := range state.Positions {
		qty := parseIntFromString(p.Quantity)
		if qty == 0 {
			continue
		}

		// Look up current price from hold overview
		holdDir := filepath.Join(root, "quote", "hold", p.Symbol)
		ov := market.ReadOverview(holdDir)
		lastPrice := 0.0
		if ov != nil {
			lastPrice = ov.Last
		}
		if lastPrice == 0 {
			// Fall back to cost price if no quote available
			lastPrice = p.CostPrice
		}

		mktValue := roundN(float64(qty)*lastPrice, 2)
		totalMktValue += mktValue

		positions[p.Symbol] = model.CurrentPosition{
			Qty:         qty,
			MarketValue: mktValue,
			AvgCost:     p.CostPrice,
			// Weight is filled in below once we know total equity
		}
	}

	// Sum USD cash (use first USD entry; fall back to first available)
	cash := 0.0
	for _, c := range state.Cash {
		if c.Currency == "USD" || c.Currency == "HKD" {
			cash += c.Available
		}
	}
	// If no USD/HKD found, use first entry
	if cash == 0 && len(state.Cash) > 0 {
		cash = state.Cash[0].Available
	}

	totalEquity := roundN(totalMktValue+cash, 2)
	cashPct := 0.0
	if totalEquity > 0 {
		cashPct = roundN(cash/totalEquity, 4)
	}

	// Compute weights now that we have totalEquity
	updatedPositions := make(map[string]model.CurrentPosition)
	for sym, pos := range positions {
		weight := 0.0
		if totalEquity > 0 {
			weight = roundN(pos.MarketValue/totalEquity, 4)
		}
		pos.Weight = weight
		updatedPositions[sym] = pos
	}

	current := model.PortfolioCurrent{
		UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
		TotalEquity: totalEquity,
		Positions:   updatedPositions,
		Cash:        roundN(cash, 2),
		CashPct:     cashPct,
	}

	data, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return err
	}
	outPath := filepath.Join(root, "portfolio", "current.json")
	return os.WriteFile(outPath, append(data, '\n'), 0644)
}

// ComputeDiff reads portfolio/target.json and portfolio/current.json,
// calculates required adjustments, and writes portfolio/diff.json.
// Returns (false, nil) if target.json does not exist.
func ComputeDiff(root string) (bool, error) {
	target, err := ParseTarget(root)
	if err != nil {
		return false, err
	}
	if target == nil {
		return false, nil // no target defined yet
	}

	// Read current portfolio
	currentData, err := os.ReadFile(filepath.Join(root, "portfolio", "current.json"))
	if err != nil {
		return false, fmt.Errorf("failed to read current.json: %w", err)
	}
	var current model.PortfolioCurrent
	if err := json.Unmarshal(currentData, &current); err != nil {
		return false, fmt.Errorf("failed to parse current.json: %w", err)
	}

	diff := computeDiff(target, &current, root)

	data, err := json.MarshalIndent(diff, "", "  ")
	if err != nil {
		return false, err
	}
	outPath := filepath.Join(root, "portfolio", "diff.json")
	if err := os.WriteFile(outPath, append(data, '\n'), 0644); err != nil {
		return false, err
	}
	return diff.RequiresRebalance, nil
}

// computeDiff calculates the adjustments needed to move from current to target.
func computeDiff(target *model.PortfolioTarget, current *model.PortfolioCurrent, root string) model.PortfolioDiff {
	diff := model.PortfolioDiff{
		ComputedAt:    time.Now().UTC().Format(time.RFC3339),
		TargetVersion: target.Version,
	}

	investableCapital := current.TotalEquity * target.TotalCapitalPct

	// Build a set of symbols we need to process (union of target and current)
	symbols := map[string]struct{}{}
	for sym := range target.Positions {
		symbols[sym] = struct{}{}
	}
	for sym := range current.Positions {
		symbols[sym] = struct{}{}
	}

	for sym := range symbols {
		targetPos, inTarget := target.Positions[sym]
		currentPos, inCurrent := current.Positions[sym]

		currentWeight := 0.0
		currentQty := int64(0)
		if inCurrent {
			currentWeight = currentPos.Weight
			currentQty = currentPos.Qty
		}

		targetWeight := 0.0
		if inTarget {
			targetWeight = targetPos.Weight
		}

		// Compute target quantity based on last price
		holdDir := filepath.Join(root, "quote", "hold", sym)
		ov := market.ReadOverview(holdDir)
		lastPrice := 0.0
		if ov != nil {
			lastPrice = ov.Last
		}
		if lastPrice == 0 && inCurrent {
			// Estimate from current market value and qty
			if currentQty > 0 {
				lastPrice = currentPos.MarketValue / float64(currentQty)
			}
		}

		targetValue := investableCapital * targetWeight
		targetQty := int64(0)
		if lastPrice > 0 {
			targetQty = int64(math.Floor(targetValue / lastPrice))
		}

		deltaQty := targetQty - currentQty
		deltaValue := roundN(float64(deltaQty)*lastPrice, 2)

		var action string
		var estimatedSide string
		switch {
		case !inTarget && inCurrent:
			action = "REMOVE"
			estimatedSide = "SELL"
		case inTarget && !inCurrent:
			action = "ADD"
			estimatedSide = "BUY"
		case deltaQty > 0:
			action = "ADD"
			estimatedSide = "BUY"
		case deltaQty < 0:
			action = "REDUCE"
			estimatedSide = "SELL"
		default:
			action = "HOLD"
			estimatedSide = ""
		}

		estimatedQty := deltaQty
		if estimatedQty < 0 {
			estimatedQty = -estimatedQty
		}

		adj := model.PortfolioAdjustment{
			Symbol:        sym,
			CurrentWeight: currentWeight,
			TargetWeight:  targetWeight,
			Action:        action,
			DeltaQty:      deltaQty,
			DeltaValue:    deltaValue,
			EstimatedSide: estimatedSide,
			EstimatedQty:  estimatedQty,
		}
		diff.Adjustments = append(diff.Adjustments, adj)

		if action != "HOLD" {
			diff.RequiresRebalance = true
		}
	}

	return diff
}

// parseIntFromString parses a string to int64, returns 0 on error.
func parseIntFromString(s string) int64 {
	var v int64
	_, err := fmt.Sscan(s, &v)
	if err != nil {
		return 0
	}
	return v
}

// roundN rounds f to n decimal places.
func roundN(f float64, n int) float64 {
	shift := math.Pow(10, float64(n))
	return math.Round(f*shift) / shift
}
