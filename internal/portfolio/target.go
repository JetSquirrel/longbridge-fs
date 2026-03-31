package portfolio

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"longbridge-fs/internal/model"
)

// ParseTarget reads portfolio/target.json and returns the parsed target portfolio.
// Returns nil, nil if the file does not exist.
func ParseTarget(root string) (*model.PortfolioTarget, error) {
	path := filepath.Join(root, "portfolio", "target.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read target.json: %w", err)
	}
	var t model.PortfolioTarget
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("failed to parse target.json: %w", err)
	}
	return &t, nil
}

// ValidateTarget checks that a PortfolioTarget is internally consistent.
func ValidateTarget(t *model.PortfolioTarget) error {
	if t == nil {
		return fmt.Errorf("target is nil")
	}
	if t.TotalCapitalPct <= 0 || t.TotalCapitalPct > 1 {
		return fmt.Errorf("total_capital_pct must be in (0, 1], got %.4f", t.TotalCapitalPct)
	}

	total := 0.0
	for sym, pos := range t.Positions {
		if pos.Weight < 0 {
			return fmt.Errorf("position %s has negative weight %.4f", sym, pos.Weight)
		}
		total += pos.Weight
	}

	// Allow a small floating-point tolerance (±1%)
	if len(t.Positions) > 0 && (total < 0.99 || total > 1.01) {
		return fmt.Errorf("position weights sum to %.4f, expected ~1.0", total)
	}

	return nil
}
