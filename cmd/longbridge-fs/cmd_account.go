package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/longportapp/openapi-go/trade"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
)

func accountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "Account and portfolio operations",
		Long:  `Query account balance, positions, and orders.`,
	}

	cmd.AddCommand(balanceCmd())
	cmd.AddCommand(positionsCmd())
	cmd.AddCommand(ordersCmd())

	return cmd
}

func balanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "balance",
		Short: "Get account balance",
		Long: `Get current account balance including cash and settled/frozen amounts.

Examples:
  longbridge-fs account balance
  longbridge-fs account balance --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBalance()
		},
	}

	return cmd
}

func runBalance() error {
	ctx := context.Background()

	tc, err := createTradeContext()
	if err != nil {
		return fmt.Errorf("failed to initialize trade context: %w", err)
	}

	// Fetch account balance
	balance, err := tc.AccountBalance(ctx, &trade.GetAccountBalance{})
	if err != nil {
		return fmt.Errorf("failed to fetch account balance: %w", err)
	}

	switch outputFormat {
	case "json":
		return outputJSON(balance)
	default:
		return outputBalanceTable(balance)
	}
}

func outputBalanceTable(balance []*trade.AccountBalance) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintln(w, "Currency\t\tTotal Cash\t\tAvailable\t\tFrozen\t\tSettling\t\tWithdrawable")
	fmt.Fprintln(w, "--------\t\t----------\t\t---------\t\t------\t\t--------\t\t------------")

	for _, acct := range balance {
		for _, cash := range acct.CashInfos {
			available := decFloat(cash.AvailableCash)
			frozen := decFloat(cash.FrozenCash)
			settling := decFloat(cash.SettlingCash)
			withdraw := decFloat(cash.WithdrawCash)

			total := available + frozen + settling
			fmt.Fprintf(w, "%s\t\t%.2f\t\t%.2f\t\t%.2f\t\t%.2f\t\t%.2f\n",
				cash.Currency,
				total,
				available,
				frozen,
				settling,
				withdraw,
			)
		}
	}

	return w.Flush()
}

func positionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "positions",
		Short: "Get stock positions",
		Long: `Get current stock positions across all accounts.

Examples:
  longbridge-fs account positions
  longbridge-fs account positions --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPositions()
		},
	}

	return cmd
}

func runPositions() error {
	ctx := context.Background()

	tc, err := createTradeContext()
	if err != nil {
		return fmt.Errorf("failed to initialize trade context: %w", err)
	}

	// Fetch stock positions
	positions, err := tc.StockPositions(ctx, []string{})
	if err != nil {
		return fmt.Errorf("failed to fetch positions: %w", err)
	}

	switch outputFormat {
	case "json":
		return outputJSON(positions)
	default:
		return outputPositionsTable(positions)
	}
}

func outputPositionsTable(positions []*trade.StockPositionChannel) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintln(w, "Symbol\t\tQuantity\t\tAvailable\t\tCost Price\t\tCurrency\t\tMarket")
	fmt.Fprintln(w, "------\t\t--------\t\t---------\t\t----------\t\t--------\t\t------")

	for _, channel := range positions {
		for _, pos := range channel.Positions {
			fmt.Fprintf(w, "%s\t\t%d\t\t%d\t\t%.3f\t\t%s\t\t%s\n",
				pos.Symbol,
				pos.Quantity,
				pos.AvailableQuantity,
				decFloat(pos.CostPrice),
				pos.Currency,
				pos.Market,
			)
		}
	}

	return w.Flush()
}

func ordersCmd() *cobra.Command {
	var today bool

	cmd := &cobra.Command{
		Use:   "orders",
		Short: "Get order history",
		Long: `Get order history. By default shows today's orders.

Examples:
  longbridge-fs account orders
  longbridge-fs account orders --today
  longbridge-fs account orders --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOrders(today)
		},
	}

	cmd.Flags().BoolVar(&today, "today", true, "Show today's orders only")

	return cmd
}

func runOrders(today bool) error {
	ctx := context.Background()

	tc, err := createTradeContext()
	if err != nil {
		return fmt.Errorf("failed to initialize trade context: %w", err)
	}

	// Fetch orders
	orders, err := tc.TodayOrders(ctx, &trade.GetTodayOrders{})
	if err != nil {
		return fmt.Errorf("failed to fetch orders: %w", err)
	}

	switch outputFormat {
	case "json":
		return outputJSON(orders)
	default:
		return outputOrdersTable(orders)
	}
}

func outputOrdersTable(orders []*trade.Order) error {
	if len(orders) == 0 {
		fmt.Println("No orders found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintln(w, "Order ID\t\tSymbol\t\tSide\t\tType\t\tQuantity\t\tPrice\t\tStatus")
	fmt.Fprintln(w, "--------\t\t------\t\t----\t\t----\t\t--------\t\t-----\t\t------")

	for _, order := range orders {
		price := "MARKET"
		if order.Price != nil {
			price = fmt.Sprintf("%.3f", decFloat(order.Price))
		}

		qty := order.Quantity

		fmt.Fprintf(w, "%s\t\t%s\t\t%s\t\t%s\t\t%s\t\t%s\t\t%s\n",
			order.OrderId,
			order.Symbol,
			order.Side,
			order.OrderType,
			qty,
			price,
			order.Status,
		)
	}

	return w.Flush()
}


// decFloat safely converts a *decimal.Decimal to float64.
func decFloat(d *decimal.Decimal) float64 {
	if d == nil {
		return 0
	}
	f, _ := d.Float64()
	return f
}
