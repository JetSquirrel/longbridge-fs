// Package portfolio implements Phase 2 of the Harness pipeline:
// L3 Portfolio Construction. It provides:
//   - SyncPortfolio: synchronises portfolio/current.json from live account state
//   - ComputeDiff: calculates portfolio/diff.json from target vs current
//   - ExecuteRebalance: converts portfolio/rebalance/pending.json into trade orders
package portfolio

import (
	"fmt"
	"log"
)

// SyncPortfolio runs the full Phase-2 portfolio synchronisation step:
//  1. SyncCurrent  – update portfolio/current.json from account state + quotes
//  2. ComputeDiff  – compute portfolio/diff.json from target vs current
//  3. ExecuteRebalance – convert pending.json into beancount ORDERs (when
//     autoRebalance is true or pending.json has auto_execute=true)
//
// It logs errors but does not propagate them so the controller can continue.
func SyncPortfolio(root string, autoRebalance bool) error {
	// Step 1: Sync current portfolio snapshot
	if err := SyncCurrent(root); err != nil {
		return fmt.Errorf("sync current: %w", err)
	}

	// Step 2: Compute diff (only if target.json exists)
	requiresRebalance, err := ComputeDiff(root)
	if err != nil {
		return fmt.Errorf("compute diff: %w", err)
	}

	// Step 3: Execute rebalance if applicable
	// Conditions: autoRebalance flag OR pending.json already exists (manual approval)
	if requiresRebalance && autoRebalance {
		log.Printf("portfolio: diff requires rebalance (auto-rebalance enabled)")
	}

	n, err := ExecuteRebalance(root)
	if err != nil {
		return fmt.Errorf("execute rebalance: %w", err)
	}
	if n > 0 {
		log.Printf("portfolio: submitted %d rebalance order(s) to trade ledger", n)
	}

	return nil
}
