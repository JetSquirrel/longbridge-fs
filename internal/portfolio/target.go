package portfolio

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"longbridge-fs/internal/model"
)

// ParseTarget reads and validates portfolio/target.json
func ParseTarget(root string) (*model.TargetPortfolio, error) {
	targetPath := filepath.Join(root, "portfolio", "target.json")
	data, err := os.ReadFile(targetPath)
	if err != nil {
		return nil, err
	}

	var target model.TargetPortfolio
	if err := json.Unmarshal(data, &target); err != nil {
		return nil, fmt.Errorf("parse target portfolio: %w", err)
	}

	if err := ValidateTarget(&target); err != nil {
		return nil, fmt.Errorf("invalid target portfolio: %w", err)
	}

	return &target, nil
}

// ValidateTarget ensures target portfolio has valid weights
func ValidateTarget(target *model.TargetPortfolio) error {
	if target.Version < 1 {
		return fmt.Errorf("version must be >= 1")
	}

	if target.TotalCapitalPct < 0 || target.TotalCapitalPct > 1 {
		return fmt.Errorf("total_capital_pct must be between 0 and 1, got %f", target.TotalCapitalPct)
	}

	if target.CashReservePct < 0 || target.CashReservePct > 1 {
		return fmt.Errorf("cash_reserve_pct must be between 0 and 1, got %f", target.CashReservePct)
	}

	// Validate that total_capital_pct + cash_reserve_pct = 1.0 (approximately)
	total := target.TotalCapitalPct + target.CashReservePct
	if total < 0.99 || total > 1.01 {
		return fmt.Errorf("total_capital_pct + cash_reserve_pct should equal 1.0, got %f", total)
	}

	// Validate position weights sum to 1.0
	var weightSum float64
	for symbol, pos := range target.Positions {
		if pos.Weight < 0 || pos.Weight > 1 {
			return fmt.Errorf("weight for %s must be between 0 and 1, got %f", symbol, pos.Weight)
		}
		weightSum += pos.Weight
	}

	if weightSum < 0.99 || weightSum > 1.01 {
		return fmt.Errorf("position weights should sum to 1.0, got %f", weightSum)
	}

	return nil
}
