package portfolio

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"longbridge-fs/internal/model"
)

// ComputeDiff calculates the difference between target and current portfolios
// and writes the result to portfolio/diff.json
func ComputeDiff(root string) error {
	// Read target portfolio
	target, err := ParseTarget(root)
	if err != nil {
		// If target doesn't exist, skip diff calculation
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read target portfolio: %w", err)
	}

	// Read current portfolio
	currentPath := filepath.Join(root, "portfolio", "current.json")
	currentData, err := os.ReadFile(currentPath)
	if err != nil {
		return fmt.Errorf("read current portfolio: %w", err)
	}

	var current model.CurrentPortfolio
	if err := json.Unmarshal(currentData, &current); err != nil {
		return fmt.Errorf("parse current portfolio: %w", err)
	}

	// Calculate diff
	diff := model.PortfolioDiff{
		ComputedAt:    time.Now().UTC().Format(time.RFC3339),
		TargetVersion: target.Version,
		Adjustments:   []model.Adjustment{},
	}

	// Build maps for easier lookup
	currentWeights := make(map[string]float64)
	for symbol, pos := range current.Positions {
		currentWeights[symbol] = pos.Weight
	}

	// Track all symbols (both target and current)
	allSymbols := make(map[string]bool)
	for symbol := range target.Positions {
		allSymbols[symbol] = true
	}
	for symbol := range current.Positions {
		allSymbols[symbol] = true
	}

	// Calculate adjustments for each symbol
	for symbol := range allSymbols {
		targetPos, hasTarget := target.Positions[symbol]
		currentPos, hasCurrent := current.Positions[symbol]

		targetWeight := 0.0
		if hasTarget {
			targetWeight = targetPos.Weight * target.TotalCapitalPct
		}

		currentWeight := 0.0
		if hasCurrent {
			currentWeight = currentPos.Weight
		}

		// Calculate target value and quantity
		targetValue := targetWeight * current.TotalEquity
		currentValue := currentWeight * current.TotalEquity

		deltaValue := targetValue - currentValue

		// Skip if difference is negligible (less than 1% of position or $100)
		threshold := math.Max(100, math.Abs(currentValue)*0.01)
		if math.Abs(deltaValue) < threshold && hasCurrent && hasTarget {
			continue
		}

		// Determine action
		action := "HOLD"
		estimatedSide := ""
		estimatedQty := int64(0)
		deltaQty := int64(0)

		if !hasTarget && hasCurrent {
			action = "CLOSE"
			estimatedSide = "SELL"
			estimatedQty = int64(currentPos.Qty)
			deltaQty = -int64(currentPos.Qty)
		} else if hasTarget && !hasCurrent {
			action = "ADD"
			estimatedSide = "BUY"
			// Estimate quantity based on current price
			if currentPrice := getSymbolPrice(root, symbol); currentPrice > 0 {
				estimatedQty = int64(targetValue / currentPrice)
				deltaQty = estimatedQty
			}
		} else if hasTarget && hasCurrent {
			if deltaValue > 0 {
				action = "REDUCE"
				estimatedSide = "BUY"
				if currentPrice := getSymbolPrice(root, symbol); currentPrice > 0 {
					estimatedQty = int64(deltaValue / currentPrice)
					deltaQty = estimatedQty
				}
			} else {
				action = "REDUCE"
				estimatedSide = "SELL"
				if currentPrice := getSymbolPrice(root, symbol); currentPrice > 0 {
					estimatedQty = int64(-deltaValue / currentPrice)
					deltaQty = -estimatedQty
				}
			}
		}

		if action != "HOLD" {
			diff.Adjustments = append(diff.Adjustments, model.Adjustment{
				Symbol:        symbol,
				CurrentWeight: currentWeight,
				TargetWeight:  targetWeight,
				Action:        action,
				DeltaQty:      deltaQty,
				DeltaValue:    deltaValue,
				EstimatedSide: estimatedSide,
				EstimatedQty:  estimatedQty,
			})
			diff.RequiresRebalance = true
		}
	}

	// Write diff.json
	data, err := json.MarshalIndent(diff, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal diff: %w", err)
	}

	return os.WriteFile(filepath.Join(root, "portfolio", "diff.json"), append(data, '\n'), 0644)
}

// getSymbolPrice retrieves current price from quote/hold/{SYMBOL}/overview.json
func getSymbolPrice(root, symbol string) float64 {
	overviewPath := filepath.Join(root, "quote", "hold", symbol, "overview.json")
	data, err := os.ReadFile(overviewPath)
	if err != nil {
		return 0
	}

	var overview model.QuoteOverview
	if err := json.Unmarshal(data, &overview); err != nil {
		return 0
	}

	return overview.Last
}
