package account

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

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

// decFloat safely converts a *decimal.Decimal to float64.
func decFloat(d *decimal.Decimal) float64 {
	if d == nil {
		return 0
	}
	f, _ := d.Float64()
	return f
}
