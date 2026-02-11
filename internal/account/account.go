package account

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"longbridge-fs/internal/market"
	"longbridge-fs/internal/model"

	"github.com/longportapp/openapi-go/trade"
	"github.com/shopspring/decimal"
)

// RefreshState fetches account balance, positions, and today's orders,
// then writes to /account/state.json.
func RefreshState(ctx context.Context, tc *trade.TradeContext, root string) error {
	state := model.AccountState{
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Account balance
	balResp, err := tc.AccountBalance(ctx, &trade.GetAccountBalance{})
	if err == nil {
		for _, ab := range balResp {
			for _, ci := range ab.CashInfos {
				state.Cash = append(state.Cash, model.CashEntry{
					Currency:  ci.Currency,
					Available: decFloat(ci.AvailableCash),
					Frozen:    decFloat(ci.FrozenCash),
					Settling:  decFloat(ci.SettlingCash),
					Withdraw:  decFloat(ci.WithdrawCash),
				})
			}
		}
	}

	// Stock positions
	posResp, err := tc.StockPositions(ctx, []string{})
	if err == nil {
		for _, ch := range posResp {
			for _, p := range ch.Positions {
				state.Positions = append(state.Positions, model.PositionEx{
					Symbol:    p.Symbol,
					Quantity:  p.Quantity,
					Available: p.AvailableQuantity,
					CostPrice: decFloat(p.CostPrice),
					Currency:  p.Currency,
					Market:    string(p.Market),
				})
			}
		}
	}

	// Today's orders
	ordResp, err := tc.TodayOrders(ctx, &trade.GetTodayOrders{})
	if err == nil {
		for _, o := range ordResp {
			state.Orders = append(state.Orders, model.OrderRef{
				OrderID: o.OrderId,
				Status:  string(o.Status),
			})
		}
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, "account", "state.json"), append(data, '\n'), 0644)
}

// GeneratePnL reads account/state.json positions, looks up current prices
// from quote/hold/{SYMBOL}/overview.json, computes unrealized P&L per position,
// and writes account/pnl.json.
func GeneratePnL(root string) error {
	// Read state.json
	stateData, err := os.ReadFile(filepath.Join(root, "account", "state.json"))
	if err != nil {
		return err
	}
	var state model.AccountState
	if err := json.Unmarshal(stateData, &state); err != nil {
		return err
	}

	report := model.PnLReport{
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	for _, pos := range state.Positions {
		qty := parseFloat(pos.Quantity)
		if qty == 0 {
			continue
		}

		// Look up current price from overview.json
		holdDir := filepath.Join(root, "quote", "hold", pos.Symbol)
		ov := market.ReadOverview(holdDir)
		lastPrice := 0.0
		if ov != nil {
			lastPrice = ov.Last
		}

		cost := qty * pos.CostPrice
		value := qty * lastPrice
		pnl := value - cost
		pnlPct := 0.0
		if cost != 0 {
			pnlPct = pnl / cost * 100
		}

		pp := model.PositionPnL{
			Symbol:    pos.Symbol,
			Quantity:  qty,
			CostPrice: pos.CostPrice,
			LastPrice: lastPrice,
			Currency:  pos.Currency,
			Cost:      roundN(cost, 2),
			Value:     roundN(value, 2),
			PnL:       roundN(pnl, 2),
			PnLPct:    roundN(pnlPct, 2),
		}
		report.Positions = append(report.Positions, pp)
		report.TotalCost += cost
		report.TotalValue += value
	}

	report.TotalPnL = roundN(report.TotalValue-report.TotalCost, 2)
	report.TotalCost = roundN(report.TotalCost, 2)
	report.TotalValue = roundN(report.TotalValue, 2)
	if report.TotalCost != 0 {
		report.TotalPct = roundN(report.TotalPnL/report.TotalCost*100, 2)
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, "account", "pnl.json"), append(data, '\n'), 0644)
}

// GeneratePortfolio aggregates all hold/ overviews + positions into
// a single /quote/portfolio.json for easy AI consumption.
func GeneratePortfolio(root string) error {
	// Read state.json for position info
	stateData, err := os.ReadFile(filepath.Join(root, "account", "state.json"))
	if err != nil {
		return err
	}
	var state model.AccountState
	if err := json.Unmarshal(stateData, &state); err != nil {
		return err
	}

	// Build position lookup map
	posMap := map[string]model.PositionEx{}
	for _, p := range state.Positions {
		posMap[p.Symbol] = p
	}

	portfolio := model.Portfolio{
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Scan all hold dirs (includes tracked quotes even without positions)
	holdBase := filepath.Join(root, "quote", "hold")
	entries, err := os.ReadDir(holdBase)
	if err != nil {
		return err
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		symbol := e.Name()
		ov := market.ReadOverview(filepath.Join(holdBase, symbol))
		if ov == nil {
			continue
		}

		pos, hasPos := posMap[symbol]
		qty := 0.0
		costPrice := 0.0
		currency := ""
		mkt := ""
		if hasPos {
			qty = parseFloat(pos.Quantity)
			costPrice = pos.CostPrice
			currency = pos.Currency
			mkt = pos.Market
		}

		pnl := 0.0
		pnlPct := 0.0
		if qty > 0 && costPrice > 0 {
			cost := qty * costPrice
			value := qty * ov.Last
			pnl = roundN(value-cost, 2)
			pnlPct = roundN((value-cost)/cost*100, 2)
		}

		portfolio.Holdings = append(portfolio.Holdings, model.PortfolioItem{
			Symbol:    symbol,
			Quantity:  qty,
			CostPrice: costPrice,
			Last:      ov.Last,
			Change:    ov.Change,
			ChangePct: ov.ChangePct,
			PnL:       pnl,
			PnLPct:    pnlPct,
			Currency:  currency,
			Market:    mkt,
		})
	}

	data, err := json.MarshalIndent(portfolio, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, "quote", "portfolio.json"), append(data, '\n'), 0644)
}

// decFloat safely converts a *decimal.Decimal to float64.
func decFloat(d *decimal.Decimal) float64 {
	if d == nil {
		return 0
	}
	f, _ := d.Float64()
	return f
}

// parseFloat parses a string to float64, returns 0 on error.
func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// roundN rounds f to n decimal places.
func roundN(f float64, n int) float64 {
	d := decimal.NewFromFloat(f)
	d = d.Round(int32(n))
	v, _ := d.Float64()
	return v
}
