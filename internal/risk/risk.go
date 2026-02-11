package risk

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"longbridge-fs/internal/market"
	"longbridge-fs/internal/model"
)

// CheckRiskRules reads /trade/risk_control.json, compares current prices
// against stop-loss/take-profit levels, and auto-appends ORDER entries
// to beancount.txt when a rule triggers.
//
// risk_control.json format:
//
//	{
//	  "700.HK":  { "stop_loss": 280.0, "take_profit": 350.0 },
//	  "AAPL.US": { "stop_loss": 150.0, "take_profit": 210.0, "qty": "10" }
//	}
func CheckRiskRules(root string) error {
	rcPath := filepath.Join(root, "trade", "risk_control.json")
	data, err := os.ReadFile(rcPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no risk control config → nothing to do
		}
		return err
	}

	var rules map[string]model.RiskRule
	if err := json.Unmarshal(data, &rules); err != nil {
		return fmt.Errorf("parse risk_control.json: %w", err)
	}

	holdBase := filepath.Join(root, "quote", "hold")
	bcPath := filepath.Join(root, "trade", "beancount.txt")

	triggered := []string{}

	for symbol, rule := range rules {
		if rule.StopLoss == 0 && rule.TakeProfit == 0 {
			continue
		}

		ov := market.ReadOverview(filepath.Join(holdBase, symbol))
		if ov == nil {
			continue // no quote yet
		}

		last := ov.Last
		if last == 0 {
			continue
		}

		var reason string
		if rule.StopLoss > 0 && last <= rule.StopLoss {
			reason = fmt.Sprintf("STOP_LOSS triggered: last=%.4f <= stop_loss=%.4f", last, rule.StopLoss)
		} else if rule.TakeProfit > 0 && last >= rule.TakeProfit {
			reason = fmt.Sprintf("TAKE_PROFIT triggered: last=%.4f >= take_profit=%.4f", last, rule.TakeProfit)
		}

		if reason == "" {
			continue
		}

		// Generate ORDER entry
		side := "SELL"
		if rule.Side != "" {
			side = strings.ToUpper(rule.Side)
		}
		qty := "ALL"
		if rule.Qty != "" {
			qty = rule.Qty
		}

		intentID := fmt.Sprintf("risk-%s-%d", strings.ReplaceAll(symbol, ".", "-"), time.Now().UnixMilli())
		date := time.Now().UTC().Format("2006-01-02")

		entry := fmt.Sprintf("\n%s ORDER\n  intent_id: %s\n  side: %s\n  symbol: %s\n  qty: %s\n  order_type: MARKET\n  tif: DAY\n  reason: %s\n",
			date, intentID, side, symbol, qty, reason)

		// Append to beancount.txt
		f, err := os.OpenFile(bcPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Printf("risk: cannot open beancount.txt: %v", err)
			continue
		}
		f.WriteString(entry)
		f.Close()

		log.Printf("risk: %s %s", symbol, reason)
		triggered = append(triggered, symbol)
	}

	// Remove triggered symbols from risk_control.json to avoid duplicate orders
	if len(triggered) > 0 {
		for _, sym := range triggered {
			delete(rules, sym)
		}
		updatedData, err := json.MarshalIndent(rules, "", "  ")
		if err == nil {
			os.WriteFile(rcPath, append(updatedData, '\n'), 0644)
		}
	}

	return nil
}
