package broker

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTWAPExecution(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	tradeDir := filepath.Join(tmpDir, "trade")
	if err := os.MkdirAll(tradeDir, 0755); err != nil {
		t.Fatalf("Failed to create trade dir: %v", err)
	}

	bcPath := filepath.Join(tradeDir, "beancount.txt")

	// Create a TWAP order
	order := `2026-03-31 * "ORDER" "BUY AAPL.US 500 via TWAP"
  ; intent_id: test-twap-001
  ; side: BUY
  ; symbol: AAPL
  ; qty: 500
  ; type: LIMIT
  ; price: 182.00
  ; tif: DAY
  ; algo: TWAP
  ; algo_duration: 1s
  ; algo_slices: 5
`
	if err := os.WriteFile(bcPath, []byte(order), 0644); err != nil {
		t.Fatalf("Failed to write order: %v", err)
	}

	// Create scheduler
	scheduler := NewAlgoScheduler(bcPath, nil, true)
	defer scheduler.Shutdown()

	ctx := context.Background()

	// Process ledger
	n, err := ProcessLedgerWithScheduler(ctx, nil, tmpDir, true, scheduler)
	if err != nil {
		t.Fatalf("ProcessLedger failed: %v", err)
	}

	if n != 1 {
		t.Errorf("Expected 1 order processed, got %d", n)
	}

	// Check that algo task was created
	activeCount := scheduler.GetActiveCount()
	if activeCount != 1 {
		t.Errorf("Expected 1 active algo task, got %d", activeCount)
	}

	// Wait for TWAP to complete
	time.Sleep(2 * time.Second)

	// Verify executions were written
	data, err := os.ReadFile(bcPath)
	if err != nil {
		t.Fatalf("Failed to read beancount: %v", err)
	}

	content := string(data)

	// Should have 5 EXECUTION entries for 5 slices
	executionCount := 0
	for i := 1; i <= 5; i++ {
		sliceLabel := "slice " + string(rune('0'+i)) + "/5"
		if contains(content, sliceLabel) {
			executionCount++
		}
	}

	if executionCount != 5 {
		t.Errorf("Expected 5 TWAP slice executions, found %d", executionCount)
		t.Logf("Beancount content:\n%s", content)
	}

	// Verify algo tasks are completed
	finalActiveCount := scheduler.GetActiveCount()
	if finalActiveCount != 0 {
		t.Errorf("Expected 0 active algo tasks after completion, got %d", finalActiveCount)
	}
}

func TestICEBERGExecution(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	tradeDir := filepath.Join(tmpDir, "trade")
	if err := os.MkdirAll(tradeDir, 0755); err != nil {
		t.Fatalf("Failed to create trade dir: %v", err)
	}

	bcPath := filepath.Join(tradeDir, "beancount.txt")

	// Create an ICEBERG order
	order := `2026-03-31 * "ORDER" "BUY TSLA.US 300 via ICEBERG"
  ; intent_id: test-iceberg-001
  ; side: BUY
  ; symbol: TSLA
  ; qty: 300
  ; type: LIMIT
  ; price: 250.00
  ; tif: DAY
  ; algo: ICEBERG
  ; algo_slices: 3
`
	if err := os.WriteFile(bcPath, []byte(order), 0644); err != nil {
		t.Fatalf("Failed to write order: %v", err)
	}

	// Create scheduler
	scheduler := NewAlgoScheduler(bcPath, nil, true)
	defer scheduler.Shutdown()

	ctx := context.Background()

	// Process ledger
	n, err := ProcessLedgerWithScheduler(ctx, nil, tmpDir, true, scheduler)
	if err != nil {
		t.Fatalf("ProcessLedger failed: %v", err)
	}

	if n != 1 {
		t.Errorf("Expected 1 order processed, got %d", n)
	}

	// Wait for ICEBERG to complete (3 slices * 2s delay)
	time.Sleep(8 * time.Second)

	// Verify executions were written
	data, err := os.ReadFile(bcPath)
	if err != nil {
		t.Fatalf("Failed to read beancount: %v", err)
	}

	content := string(data)

	// Should have 3 EXECUTION entries for 3 slices
	executionCount := 0
	for i := 1; i <= 3; i++ {
		sliceLabel := "slice " + string(rune('0'+i)) + "/3"
		if contains(content, sliceLabel) {
			executionCount++
		}
	}

	if executionCount != 3 {
		t.Errorf("Expected 3 ICEBERG slice executions, found %d", executionCount)
		t.Logf("Beancount content:\n%s", content)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || contains(s[1:], substr)))
}
