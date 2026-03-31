package portfolio

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"longbridge-fs/internal/model"
)

// AutoCreatePending reads portfolio/diff.json and, when requires_rebalance is true,
// automatically generates portfolio/rebalance/pending.json so the controller can
// process it on the next cycle. It is a no-op when pending.json already exists.
func AutoCreatePending(root string) error {
	// Read diff.json
	diffPath := filepath.Join(root, "portfolio", "diff.json")
	data, err := os.ReadFile(diffPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No diff available
		}
		return fmt.Errorf("read diff: %w", err)
	}

	var diff model.PortfolioDiff
	if err := json.Unmarshal(data, &diff); err != nil {
		return fmt.Errorf("parse diff: %w", err)
	}

	if !diff.RequiresRebalance || len(diff.Adjustments) == 0 {
		return nil // Nothing to do
	}

	// Don't overwrite an existing pending.json
	pendingPath := filepath.Join(root, "portfolio", "rebalance", "pending.json")
	if _, err := os.Stat(pendingPath); err == nil {
		return nil // Already pending
	}

	// Convert diff adjustments to rebalance orders
	rebalanceID := fmt.Sprintf("rebal-%s", time.Now().UTC().Format("20060102-150405"))
	pending := model.RebalancePending{
		RebalanceID: rebalanceID,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		CreatedBy:   "controller-auto-rebalance",
		AutoExecute: true,
		Orders:      make([]model.RebalanceOrder, 0, len(diff.Adjustments)),
	}

	for _, adj := range diff.Adjustments {
		if adj.EstimatedQty <= 0 {
			continue
		}

		order := model.RebalanceOrder{
			Symbol: adj.Symbol,
			Side:   adj.EstimatedSide,
			Qty:    adj.EstimatedQty,
			Type:   "MARKET",
			TIF:    "DAY",
		}

		pending.Orders = append(pending.Orders, order)
	}

	if len(pending.Orders) == 0 {
		return nil
	}

	// Write pending.json
	rebalDir := filepath.Join(root, "portfolio", "rebalance")
	if err := os.MkdirAll(rebalDir, 0755); err != nil {
		return fmt.Errorf("create rebalance dir: %w", err)
	}

	out, err := json.MarshalIndent(pending, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal pending: %w", err)
	}

	return os.WriteFile(pendingPath, append(out, '\n'), 0644)
}

// ProcessRebalance reads portfolio/rebalance/pending.json and converts it to ORDER entries
// in trade/beancount.txt, then archives the pending file to history/
func ProcessRebalance(root string) error {
	pendingPath := filepath.Join(root, "portfolio", "rebalance", "pending.json")

	// Check if pending.json exists
	data, err := os.ReadFile(pendingPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No pending rebalance
		}
		return fmt.Errorf("read pending rebalance: %w", err)
	}

	var pending model.RebalancePending
	if err := json.Unmarshal(data, &pending); err != nil {
		return fmt.Errorf("parse pending rebalance: %w", err)
	}

	// Convert to ORDER entries
	if err := writeRebalanceOrders(root, &pending); err != nil {
		return fmt.Errorf("write rebalance orders: %w", err)
	}

	// Archive pending.json to history
	if err := archivePending(root, &pending); err != nil {
		return fmt.Errorf("archive pending: %w", err)
	}

	// Remove pending.json
	if err := os.Remove(pendingPath); err != nil {
		return fmt.Errorf("remove pending: %w", err)
	}

	return nil
}

// writeRebalanceOrders appends ORDER entries to trade/beancount.txt
func writeRebalanceOrders(root string, pending *model.RebalancePending) error {
	beancountPath := filepath.Join(root, "trade", "beancount.txt")

	f, err := os.OpenFile(beancountPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	timestamp := time.Now().UTC().Format("2006-01-02")

	for i, order := range pending.Orders {
		intentID := fmt.Sprintf("%s-%03d", strings.ReplaceAll(pending.RebalanceID, "rebal-", ""), i+1)

		var orderLines strings.Builder
		orderLines.WriteString("\n")
		orderLines.WriteString(fmt.Sprintf("%s * \"ORDER\" \"%s %d %s %s\"\n",
			timestamp, order.Side, order.Qty, order.Symbol, pending.RebalanceID))
		orderLines.WriteString(fmt.Sprintf("  ; intent_id: %s\n", intentID))
		orderLines.WriteString(fmt.Sprintf("  ; side: %s\n", order.Side))
		orderLines.WriteString(fmt.Sprintf("  ; symbol: %s\n", order.Symbol))
		orderLines.WriteString(fmt.Sprintf("  ; qty: %d\n", order.Qty))
		orderLines.WriteString(fmt.Sprintf("  ; type: %s\n", order.Type))
		if order.Price > 0 {
			orderLines.WriteString(fmt.Sprintf("  ; price: %.2f\n", order.Price))
		}
		orderLines.WriteString(fmt.Sprintf("  ; tif: %s\n", order.TIF))
		orderLines.WriteString("  ; source: rebalance\n")
		orderLines.WriteString(fmt.Sprintf("  ; rebalance_id: %s\n", pending.RebalanceID))

		if _, err := f.WriteString(orderLines.String()); err != nil {
			return err
		}
	}

	return nil
}

// archivePending saves the pending rebalance to history with timestamp
func archivePending(root string, pending *model.RebalancePending) error {
	historyDir := filepath.Join(root, "portfolio", "history")
	if err := os.MkdirAll(historyDir, 0755); err != nil {
		return err
	}

	timestamp := time.Now().UTC().Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s.json", timestamp, pending.RebalanceID)
	historyPath := filepath.Join(historyDir, filename)

	data, err := json.MarshalIndent(pending, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(historyPath, append(data, '\n'), 0644)
}

// ArchiveTarget saves the current target.json to history/ with a timestamp
func ArchiveTarget(root string) error {
	targetPath := filepath.Join(root, "portfolio", "target.json")
	data, err := os.ReadFile(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No target to archive
		}
		return err
	}

	historyDir := filepath.Join(root, "portfolio", "history")
	if err := os.MkdirAll(historyDir, 0755); err != nil {
		return err
	}

	timestamp := time.Now().UTC().Format("20060102-150405")
	historyPath := filepath.Join(historyDir, fmt.Sprintf("%s-target.json", timestamp))

	return os.WriteFile(historyPath, data, 0644)
}
