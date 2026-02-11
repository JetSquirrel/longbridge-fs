package broker

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"longbridge-fs/internal/ledger"
	"longbridge-fs/internal/model"

	"github.com/longportapp/openapi-go/trade"
	"github.com/shopspring/decimal"
)

// ProcessLedger reads the beancount ledger, finds unprocessed ORDER entries,
// and executes them. Returns the number of new executions.
func ProcessLedger(ctx context.Context, tc *trade.TradeContext, root string, useMock bool) (int, error) {
	bcPath := filepath.Join(root, "trade", "beancount.txt")
	entries, err := ledger.ParseEntries(bcPath)
	if err != nil {
		return 0, err
	}

	processed, orders := ledger.BuildLedgerState(entries)
	executed := 0

	for _, oe := range orders {
		o := ledger.OrderFromEntry(oe)
		if o.IntentID == "" {
			continue
		}
		if processed[o.IntentID] {
			continue
		}

		// Handle CANCEL action
		if strings.ToUpper(o.Side) == "" && oe.Meta["action"] == "CANCEL" {
			orderID := oe.Meta["order_id"]
			if orderID == "" {
				continue
			}
			if !useMock && tc != nil {
				if err := tc.CancelOrder(ctx, orderID); err != nil {
					AppendRejection(bcPath, o.IntentID, ledger.FullSymbol(o.Symbol, o.Market), o.Side, o.Qty, err.Error())
				} else {
					AppendExecution(bcPath, o.IntentID, "CANCEL-"+orderID, ledger.FullSymbol(o.Symbol, o.Market), "", "", "0")
					log.Printf("cancelled order: intent=%s order_id=%s", o.IntentID, orderID)
				}
			}
			processed[o.IntentID] = true
			executed++
			continue
		}

		sym := ledger.FullSymbol(o.Symbol, o.Market)

		if useMock {
			orderID, price := ExecuteOrderMock(o)
			AppendExecution(bcPath, o.IntentID, orderID, sym, o.Side, price, o.Qty)
			log.Printf("mock execution: intent=%s -> %s", o.IntentID, orderID)
		} else if tc != nil {
			orderID, err := ExecuteOrder(ctx, tc, o)
			if err != nil {
				AppendRejection(bcPath, o.IntentID, sym, o.Side, o.Qty, err.Error())
				log.Printf("order rejected: intent=%s err=%v", o.IntentID, err)
			} else {
				AppendExecution(bcPath, o.IntentID, orderID, sym, o.Side, o.Price, o.Qty)
				log.Printf("order submitted: intent=%s -> %s", o.IntentID, orderID)
			}
		}

		processed[o.IntentID] = true
		executed++
	}

	return executed, nil
}

// ExecuteOrder submits an order via the Longbridge SDK.
func ExecuteOrder(ctx context.Context, tc *trade.TradeContext, o model.ParsedOrder) (string, error) {
	sym := ledger.FullSymbol(o.Symbol, o.Market)

	qty, err := strconv.ParseUint(o.Qty, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid qty %q: %w", o.Qty, err)
	}

	req := &trade.SubmitOrder{
		Symbol:            sym,
		OrderType:         MapOrderType(o.OrderType),
		Side:              MapOrderSide(o.Side),
		SubmittedQuantity: qty,
		TimeInForce:       MapTimeInForce(o.TIF),
		Remark:            "longbridge-fs:" + o.IntentID,
	}

	if o.Price != "" {
		p, err := decimal.NewFromString(o.Price)
		if err == nil {
			req.SubmittedPrice = p
		}
	}

	orderID, err := tc.SubmitOrder(ctx, req)
	if err != nil {
		return "", err
	}
	return orderID, nil
}

// ExecuteOrderMock returns a mock order ID and price.
func ExecuteOrderMock(o model.ParsedOrder) (orderID string, price string) {
	orderID = fmt.Sprintf("LOCAL-%d", time.Now().UnixNano())
	price = o.Price
	if price == "" {
		price = "100.00" // mock market price
	}
	return
}

// AppendExecution appends an EXECUTION entry to the beancount ledger.
func AppendExecution(bcPath, intentID, orderID, symbol, side, price, qty string) {
	f, err := os.OpenFile(bcPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("append execution failed: %v", err)
		return
	}
	defer f.Close()

	date := time.Now().Format("2006-01-02")
	text := fmt.Sprintf("\n%s * \"EXECUTION\" \"%s %s\"\n", date, side, symbol)
	text += fmt.Sprintf("  ; intent_id: %s\n", intentID)
	text += fmt.Sprintf("  ; order_id: %s\n", orderID)
	text += fmt.Sprintf("  ; status: FILLED\n")
	text += fmt.Sprintf("  ; symbol: %s\n", symbol)
	text += fmt.Sprintf("  ; side: %s\n", side)
	text += fmt.Sprintf("  ; qty: %s\n", qty)
	if price != "" {
		text += fmt.Sprintf("  ; price: %s\n", price)
	}
	text += "\n"
	f.WriteString(text)
}

// AppendRejection appends a REJECTION entry to the beancount ledger.
func AppendRejection(bcPath, intentID, symbol, side, qty, reason string) {
	f, err := os.OpenFile(bcPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("append rejection failed: %v", err)
		return
	}
	defer f.Close()

	date := time.Now().Format("2006-01-02")
	text := fmt.Sprintf("\n%s * \"REJECTION\" \"%s %s\"\n", date, side, symbol)
	text += fmt.Sprintf("  ; intent_id: %s\n", intentID)
	text += fmt.Sprintf("  ; status: REJECTED\n")
	text += fmt.Sprintf("  ; reason: %s\n", reason)
	text += fmt.Sprintf("  ; symbol: %s\n", symbol)
	text += fmt.Sprintf("  ; side: %s\n", side)
	text += fmt.Sprintf("  ; qty: %s\n", qty)
	text += "\n"
	f.WriteString(text)
}

// MapOrderType converts string to SDK OrderType.
func MapOrderType(s string) trade.OrderType {
	switch strings.ToUpper(s) {
	case "LIMIT", "LO":
		return trade.OrderType("LO")
	case "MARKET", "MO":
		return trade.OrderType("MO")
	case "ELO":
		return trade.OrderType("ELO")
	case "ALO":
		return trade.OrderType("ALO")
	default:
		return trade.OrderType("MO")
	}
}

// MapOrderSide converts string to SDK OrderSide.
func MapOrderSide(s string) trade.OrderSide {
	switch strings.ToUpper(s) {
	case "BUY":
		return trade.OrderSide("Buy")
	case "SELL":
		return trade.OrderSide("Sell")
	default:
		return trade.OrderSide("Buy")
	}
}

// MapTimeInForce converts string to SDK TimeType.
func MapTimeInForce(s string) trade.TimeType {
	switch strings.ToUpper(s) {
	case "DAY":
		return trade.TimeType("Day")
	case "GTC":
		return trade.TimeType("GoodTilCanceled")
	case "GTD":
		return trade.TimeType("GoodTilDate")
	default:
		return trade.TimeType("Day")
	}
}
