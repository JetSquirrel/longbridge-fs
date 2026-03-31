package portfolio

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"longbridge-fs/internal/model"
)

func TestValidateTarget(t *testing.T) {
	tests := []struct {
		name    string
		target  *model.PortfolioTarget
		wantErr bool
	}{
		{
			name: "valid target",
			target: &model.PortfolioTarget{
				Version:         1,
				TotalCapitalPct: 0.80,
				Positions: map[string]model.TargetPosition{
					"AAPL.US": {Weight: 0.50},
					"TSLA.US": {Weight: 0.50},
				},
				CashReservePct: 0.20,
			},
			wantErr: false,
		},
		{
			name: "empty positions (all cash)",
			target: &model.PortfolioTarget{
				Version:         1,
				TotalCapitalPct: 0.80,
				Positions:       map[string]model.TargetPosition{},
				CashReservePct:  0.20,
			},
			wantErr: false,
		},
		{
			name: "weights don't sum to 1",
			target: &model.PortfolioTarget{
				Version:         1,
				TotalCapitalPct: 0.80,
				Positions: map[string]model.TargetPosition{
					"AAPL.US": {Weight: 0.30},
					"TSLA.US": {Weight: 0.30},
				},
				CashReservePct: 0.20,
			},
			wantErr: true,
		},
		{
			name: "invalid capital pct",
			target: &model.PortfolioTarget{
				Version:         1,
				TotalCapitalPct: 0.0,
				Positions:       map[string]model.TargetPosition{},
			},
			wantErr: true,
		},
		{
			name:    "nil target",
			target:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTarget(tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTarget() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseTarget(t *testing.T) {
	dir := t.TempDir()
	portfolioDir := filepath.Join(dir, "portfolio")
	if err := os.MkdirAll(portfolioDir, 0755); err != nil {
		t.Fatal(err)
	}

	t.Run("missing file returns nil", func(t *testing.T) {
		got, err := ParseTarget(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Errorf("expected nil, got %+v", got)
		}
	})

	t.Run("valid file parses correctly", func(t *testing.T) {
		target := model.PortfolioTarget{
			Version:         1,
			TotalCapitalPct: 0.80,
			Positions: map[string]model.TargetPosition{
				"AAPL.US": {Weight: 1.0, Reason: "test"},
			},
			CashReservePct: 0.20,
		}
		data, _ := json.MarshalIndent(target, "", "  ")
		if err := os.WriteFile(filepath.Join(portfolioDir, "target.json"), data, 0644); err != nil {
			t.Fatal(err)
		}

		got, err := ParseTarget(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		if got.Version != 1 {
			t.Errorf("version = %d, want 1", got.Version)
		}
		if got.TotalCapitalPct != 0.80 {
			t.Errorf("total_capital_pct = %.2f, want 0.80", got.TotalCapitalPct)
		}
		if _, ok := got.Positions["AAPL.US"]; !ok {
			t.Error("expected AAPL.US position")
		}
	})
}

func TestComputeDiff_NoTarget(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "portfolio"), 0755); err != nil {
		t.Fatal(err)
	}

	// Write a current.json but no target.json
	current := model.PortfolioCurrent{
		UpdatedAt:   "2026-01-01T00:00:00Z",
		TotalEquity: 100000.0,
		Positions:   map[string]model.CurrentPosition{},
		Cash:        100000.0,
		CashPct:     1.0,
	}
	data, _ := json.MarshalIndent(current, "", "  ")
	os.WriteFile(filepath.Join(dir, "portfolio", "current.json"), data, 0644)

	requires, err := ComputeDiff(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if requires {
		t.Error("expected requires=false when no target.json")
	}
}

func TestSyncCurrent(t *testing.T) {
	dir := t.TempDir()

	// Create required directories
	os.MkdirAll(filepath.Join(dir, "account"), 0755)
	os.MkdirAll(filepath.Join(dir, "portfolio"), 0755)
	os.MkdirAll(filepath.Join(dir, "quote", "hold"), 0755)

	// Write account state with no positions and some cash
	state := model.AccountState{
		UpdatedAt: "2026-01-01T00:00:00Z",
		Cash: []model.CashEntry{
			{Currency: "USD", Available: 50000.0},
		},
		Positions: []model.PositionEx{},
	}
	stateData, _ := json.MarshalIndent(state, "", "  ")
	os.WriteFile(filepath.Join(dir, "account", "state.json"), stateData, 0644)

	if err := SyncCurrent(dir); err != nil {
		t.Fatalf("SyncCurrent failed: %v", err)
	}

	// Read and verify output
	outData, err := os.ReadFile(filepath.Join(dir, "portfolio", "current.json"))
	if err != nil {
		t.Fatalf("current.json not written: %v", err)
	}
	var current model.PortfolioCurrent
	if err := json.Unmarshal(outData, &current); err != nil {
		t.Fatalf("failed to parse current.json: %v", err)
	}
	if current.TotalEquity != 50000.0 {
		t.Errorf("total_equity = %.2f, want 50000.0", current.TotalEquity)
	}
	if current.Cash != 50000.0 {
		t.Errorf("cash = %.2f, want 50000.0", current.Cash)
	}
	if current.CashPct != 1.0 {
		t.Errorf("cash_pct = %.4f, want 1.0", current.CashPct)
	}
}

func TestExecuteRebalance_NoPending(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "portfolio", "rebalance"), 0755)
	os.MkdirAll(filepath.Join(dir, "trade"), 0755)

	// Write empty beancount.txt
	os.WriteFile(filepath.Join(dir, "trade", "beancount.txt"), []byte("; ledger\n"), 0644)

	n, err := ExecuteRebalance(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 orders, got %d", n)
	}
}

func TestExecuteRebalance_WithPending(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "portfolio", "rebalance"), 0755)
	os.MkdirAll(filepath.Join(dir, "portfolio", "history"), 0755)
	os.MkdirAll(filepath.Join(dir, "trade"), 0755)

	// Write beancount.txt
	os.WriteFile(filepath.Join(dir, "trade", "beancount.txt"), []byte("; ledger\n"), 0644)

	// Write pending.json
	pending := model.RebalancePending{
		RebalanceID: "rebal-20260101-001",
		CreatedAt:   "2026-01-01T00:00:00Z",
		AutoExecute: false,
		Orders: []model.RebalanceOrder{
			{Symbol: "AAPL.US", Side: "BUY", Qty: 10, Type: "LIMIT", Price: 180.00, TIF: "DAY"},
			{Symbol: "TSLA.US", Side: "SELL", Qty: 5, Type: "LIMIT", Price: 250.00, TIF: "DAY"},
		},
	}
	pendingData, _ := json.MarshalIndent(pending, "", "  ")
	os.WriteFile(filepath.Join(dir, "portfolio", "rebalance", "pending.json"), pendingData, 0644)

	n, err := ExecuteRebalance(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 orders, got %d", n)
	}

	// Verify pending.json was removed
	if _, err := os.Stat(filepath.Join(dir, "portfolio", "rebalance", "pending.json")); !os.IsNotExist(err) {
		t.Error("pending.json should be removed after execution")
	}

	// Verify beancount.txt was updated
	bcData, _ := os.ReadFile(filepath.Join(dir, "trade", "beancount.txt"))
	bcContent := string(bcData)
	if len(bcContent) <= len("; ledger\n") {
		t.Error("beancount.txt should have order entries appended")
	}
	// Check for ORDER entries
	if !strings.Contains(bcContent, "ORDER") {
		t.Error("expected ORDER entries in beancount.txt")
	}
	if !strings.Contains(bcContent, "source: rebalance") {
		t.Error("expected 'source: rebalance' metadata in ORDER entries")
	}
	if !strings.Contains(bcContent, pending.RebalanceID) {
		t.Errorf("expected rebalance_id %s in ORDER entries", pending.RebalanceID)
	}

	// Verify history file was created
	histEntries, _ := os.ReadDir(filepath.Join(dir, "portfolio", "history"))
	if len(histEntries) != 1 {
		t.Errorf("expected 1 history file, got %d", len(histEntries))
	}
}
