package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/longbridge/openapi-go/trade"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
)

func orderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "order",
		Aliases: []string{"trade"},
		Short:   "Trading operations",
		Long:    `Submit, cancel, and modify orders.`,
	}

	cmd.AddCommand(submitOrderCmd())
	cmd.AddCommand(cancelOrderCmd())

	return cmd
}

func submitOrderCmd() *cobra.Command {
	var (
		orderType string
		price     string
		tif       string
		remark    string
	)

	cmd := &cobra.Command{
		Use:   "submit [symbol] [side] [quantity]",
		Short: "Submit a new order",
		Long: `Submit a new order to buy or sell a security.

Side: BUY or SELL
Order Type: MARKET, LIMIT, ELO, ALO (default: MARKET)
Time In Force: DAY, GTC, GTD (default: DAY)

Examples:
  longbridge-fs order submit AAPL.US BUY 100
  longbridge-fs order submit TSLA.US BUY 50 --type LIMIT --price 180.50
  longbridge-fs order submit 700.HK SELL 1000 --type LIMIT --price 350.00 --tif GTC`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 3 {
				return fmt.Errorf("requires exactly 3 arguments: symbol, side, quantity")
			}

			symbol := args[0]
			side := strings.ToUpper(args[1])
			qty, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid quantity: %w", err)
			}

			return runSubmitOrder(symbol, side, qty, orderType, price, tif, remark)
		},
	}

	cmd.Flags().StringVar(&orderType, "type", "MARKET", "Order type (MARKET, LIMIT, ELO, ALO)")
	cmd.Flags().StringVar(&price, "price", "", "Limit price (required for LIMIT orders)")
	cmd.Flags().StringVar(&tif, "tif", "DAY", "Time in force (DAY, GTC, GTD)")
	cmd.Flags().StringVar(&remark, "remark", "", "Order remark/comment")

	return cmd
}

func runSubmitOrder(symbol, sideStr string, qty uint64, orderTypeStr, priceStr, tifStr, remark string) error {
	ctx := context.Background()

	tc, err := createTradeContext()
	if err != nil {
		return fmt.Errorf("failed to initialize trade context: %w", err)
	}

	// Map order type
	var orderType trade.OrderType
	switch strings.ToUpper(orderTypeStr) {
	case "LIMIT", "LO":
		orderType = trade.OrderType("LO")
	case "MARKET", "MO":
		orderType = trade.OrderType("MO")
	case "ELO":
		orderType = trade.OrderType("ELO")
	case "ALO":
		orderType = trade.OrderType("ALO")
	default:
		orderType = trade.OrderType("MO")
	}

	// Map side
	var side trade.OrderSide
	switch strings.ToUpper(sideStr) {
	case "BUY", "B":
		side = trade.OrderSide("Buy")
	case "SELL", "S":
		side = trade.OrderSide("Sell")
	default:
		return fmt.Errorf("invalid side: %s (must be BUY or SELL)", sideStr)
	}

	// Map time in force
	var tif trade.TimeType
	switch strings.ToUpper(tifStr) {
	case "DAY":
		tif = trade.TimeType("Day")
	case "GTC":
		tif = trade.TimeType("GoodTilCanceled")
	case "GTD":
		tif = trade.TimeType("GoodTilDate")
	default:
		tif = trade.TimeType("Day")
	}

	// Build order request
	order := &trade.SubmitOrder{
		Symbol:            symbol,
		OrderType:         orderType,
		Side:              side,
		SubmittedQuantity: qty,
		TimeInForce:       tif,
		Remark:            remark,
	}

	// Add price for limit orders
	if orderType == trade.OrderType("LO") || orderType == trade.OrderType("ELO") || orderType == trade.OrderType("ALO") {
		if priceStr == "" {
			return fmt.Errorf("price is required for %s orders", orderTypeStr)
		}
		price, err := decimal.NewFromString(priceStr)
		if err != nil {
			return fmt.Errorf("invalid price: %w", err)
		}
		order.SubmittedPrice = price
	}

	// Submit order
	orderID, err := tc.SubmitOrder(ctx, order)
	if err != nil {
		return fmt.Errorf("failed to submit order: %w", err)
	}

	// Output result
	if outputFormat == "json" {
		return outputJSON(map[string]interface{}{
			"order_id": orderID,
			"symbol":   symbol,
			"side":     sideStr,
			"quantity": qty,
			"type":     orderTypeStr,
			"status":   "submitted",
		})
	}

	fmt.Printf("✓ Order submitted successfully\n")
	fmt.Printf("Order ID: %s\n", orderID)
	fmt.Printf("Symbol: %s\n", symbol)
	fmt.Printf("Side: %s\n", sideStr)
	fmt.Printf("Quantity: %d\n", qty)
	fmt.Printf("Type: %s\n", orderTypeStr)
	if priceStr != "" {
		fmt.Printf("Price: %s\n", priceStr)
	}
	fmt.Printf("Time In Force: %s\n", tifStr)

	return nil
}

func cancelOrderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel [order-id]",
		Short: "Cancel an existing order",
		Long: `Cancel an existing order by its order ID.

Examples:
  longbridge-fs order cancel 1234567890
  longbridge-fs order cancel 9876543210 --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("requires exactly 1 argument: order-id")
			}
			return runCancelOrder(args[0])
		},
	}

	return cmd
}

func runCancelOrder(orderID string) error {
	ctx := context.Background()

	tc, err := createTradeContext()
	if err != nil {
		return fmt.Errorf("failed to initialize trade context: %w", err)
	}

	// Cancel order
	err = tc.CancelOrder(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to cancel order: %w", err)
	}

	// Output result
	if outputFormat == "json" {
		return outputJSON(map[string]interface{}{
			"order_id": orderID,
			"status":   "cancelled",
		})
	}

	fmt.Printf("✓ Order cancelled successfully\n")
	fmt.Printf("Order ID: %s\n", orderID)

	return nil
}
