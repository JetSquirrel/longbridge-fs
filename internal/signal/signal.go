package signal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"longbridge-fs/internal/model"
)

// ComputeAll scans signal/definitions/, computes builtin signals from quote data,
// writes per-symbol output files, and regenerates signal/active.json.
func ComputeAll(root string) error {
	defs, err := ListDefinitions(root)
	if err != nil {
		return fmt.Errorf("list signal definitions: %w", err)
	}

	if len(defs) == 0 {
		return nil // Nothing to compute
	}

	active := &model.ActiveSignals{
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Signals:   []model.ActiveSignalEntry{},
	}

	// Group definitions by symbol so we load kline data once per symbol
	symbolDefs := make(map[string][]*model.SignalDefinition)
	for _, def := range defs {
		for _, sym := range def.Symbols {
			symbolDefs[sym] = append(symbolDefs[sym], def)
		}
	}

	now := active.UpdatedAt

	for symbol, sdefs := range symbolDefs {
		// Load kline data (daily close prices)
		prices, err := loadClosePrices(root, symbol)
		if err != nil {
			fmt.Printf("Warning: skipping signal computation for %s: %v\n", symbol, err)
			continue
		}

		output := &model.SignalOutput{
			Symbol:    symbol,
			UpdatedAt: now,
			Signals:   []model.SignalEntry{},
		}

		for _, def := range sdefs {
			if def.Type != "builtin" {
				continue // External signals are written by agents, not computed here
			}

			entry, err := computeBuiltin(def, prices, now)
			if err != nil {
				fmt.Printf("Warning: failed to compute signal %s for %s: %v\n", def.Name, symbol, err)
				continue
			}

			output.Signals = append(output.Signals, *entry)

			active.Signals = append(active.Signals, model.ActiveSignalEntry{
				Symbol:   symbol,
				Name:     def.Name,
				Value:    entry.Value,
				Strength: entry.Strength,
			})
		}

		if err := WriteOutput(root, symbol, output); err != nil {
			fmt.Printf("Warning: failed to write signal output for %s: %v\n", symbol, err)
		}
		if err := AppendHistory(root, symbol, output); err != nil {
			fmt.Printf("Warning: failed to append signal history for %s: %v\n", symbol, err)
		}
	}

	return WriteActiveSignals(root, active)
}

// computeBuiltin dispatches to the appropriate indicator function based on the definition params.
func computeBuiltin(def *model.SignalDefinition, prices []float64, now string) (*model.SignalEntry, error) {
	indicator, _ := def.Params["indicator"].(string)

	var value, detail string
	var strength float64

	switch indicator {
	case "SMA_CROSS":
		fastPeriod := paramInt(def.Params, "fast_period", 5)
		slowPeriod := paramInt(def.Params, "slow_period", 20)
		value, strength, detail = ComputeSMACross(prices, fastPeriod, slowPeriod)

	case "RSI":
		period := paramInt(def.Params, "period", 14)
		overbought := paramFloat(def.Params, "overbought", 70)
		oversold := paramFloat(def.Params, "oversold", 30)
		value, strength, detail = ComputeRSI(prices, period, overbought, oversold)

	case "PRICE_CHANGE":
		thresholdPct := paramFloat(def.Params, "threshold_pct", 5.0)
		window := paramInt(def.Params, "window", 5)
		value, strength, detail = ComputePriceChange(prices, thresholdPct, window)

	default:
		return nil, fmt.Errorf("unknown indicator %q", indicator)
	}

	return &model.SignalEntry{
		Name:       def.Name,
		Value:      value,
		Strength:   strength,
		Detail:     detail,
		ComputedAt: now,
	}, nil
}

// loadClosePrices reads the daily candlestick JSON for a symbol and returns close prices.
func loadClosePrices(root, symbol string) ([]float64, error) {
	klinePath := filepath.Join(root, "quote", "hold", symbol, "D.json")
	data, err := os.ReadFile(klinePath)
	if err != nil {
		return nil, fmt.Errorf("read kline data: %w", err)
	}

	var sticks []model.Candlestick
	if err := json.Unmarshal(data, &sticks); err != nil {
		return nil, fmt.Errorf("parse kline data: %w", err)
	}

	if len(sticks) == 0 {
		return nil, fmt.Errorf("no kline data available")
	}

	prices := make([]float64, len(sticks))
	for i, s := range sticks {
		prices[i] = s.Close
	}

	return prices, nil
}
