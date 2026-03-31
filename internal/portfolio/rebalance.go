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

// ExecuteRebalance reads portfolio/rebalance/pending.json, converts the orders
// into beancount ORDER entries, appends them to trade/beancount.txt, then
// archives the pending file to portfolio/history/{timestamp}_{id}.json and
// removes the pending file.
//
// Returns (0, nil) if no pending.json exists.
func ExecuteRebalance(root string) (int, error) {
	pendingPath := filepath.Join(root, "portfolio", "rebalance", "pending.json")
	data, err := os.ReadFile(pendingPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to read pending.json: %w", err)
	}

	var pending model.RebalancePending
	if err := json.Unmarshal(data, &pending); err != nil {
		return 0, fmt.Errorf("failed to parse pending.json: %w", err)
	}

	if len(pending.Orders) == 0 {
		return 0, nil
	}

	bcPath := filepath.Join(root, "trade", "beancount.txt")
	if err := appendRebalanceOrders(bcPath, &pending); err != nil {
		return 0, fmt.Errorf("failed to append orders to beancount: %w", err)
	}

	// Archive pending to history
	if err := archivePending(root, &pending, data); err != nil {
		// Log but don't fail – orders already written
		fmt.Printf("WARNING: failed to archive pending.json: %v\n", err)
	}

	// Remove pending file
	if err := os.Remove(pendingPath); err != nil && !os.IsNotExist(err) {
		fmt.Printf("WARNING: failed to remove pending.json: %v\n", err)
	}

	return len(pending.Orders), nil
}

// appendRebalanceOrders writes beancount ORDER entries for each order in the
// RebalancePending to the beancount ledger.
func appendRebalanceOrders(bcPath string, pending *model.RebalancePending) error {
	f, err := os.OpenFile(bcPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	date := time.Now().Format("2006-01-02")
	for i, o := range pending.Orders {
		// Compact intent_id: date digits + sequential index
		intentID := fmt.Sprintf("%s%03d", strings.ReplaceAll(date, "-", ""), i+1)

		priceStr := ""
		if o.Price > 0 {
			priceStr = fmt.Sprintf("%.4f", o.Price)
		}
		tif := o.TIF
		if tif == "" {
			tif = "DAY"
		}
		orderType := o.Type
		if orderType == "" {
			if o.Price > 0 {
				orderType = "LIMIT"
			} else {
				orderType = "MARKET"
			}
		}

		desc := fmt.Sprintf("%s %s %d via REBALANCE", o.Side, o.Symbol, o.Qty)
		entry := fmt.Sprintf("\n%s * \"ORDER\" \"%s\"\n", date, desc)
		entry += fmt.Sprintf("  ; intent_id: %s\n", intentID)
		entry += fmt.Sprintf("  ; side: %s\n", o.Side)
		entry += fmt.Sprintf("  ; symbol: %s\n", o.Symbol)
		entry += fmt.Sprintf("  ; qty: %d\n", o.Qty)
		entry += fmt.Sprintf("  ; type: %s\n", orderType)
		if priceStr != "" {
			entry += fmt.Sprintf("  ; price: %s\n", priceStr)
		}
		entry += fmt.Sprintf("  ; tif: %s\n", tif)
		entry += fmt.Sprintf("  ; source: rebalance\n")
		entry += fmt.Sprintf("  ; rebalance_id: %s\n", pending.RebalanceID)
		entry += "\n"

		if _, err := f.WriteString(entry); err != nil {
			return err
		}
	}
	return nil
}

// archivePending saves the raw pending JSON into portfolio/history/.
func archivePending(root string, pending *model.RebalancePending, raw []byte) error {
	histDir := filepath.Join(root, "portfolio", "history")
	if err := os.MkdirAll(histDir, 0755); err != nil {
		return err
	}
	ts := time.Now().UTC().Format("20060102T150405Z")
	name := fmt.Sprintf("%s_%s.json", ts, pending.RebalanceID)
	return os.WriteFile(filepath.Join(histDir, name), raw, 0644)
}
