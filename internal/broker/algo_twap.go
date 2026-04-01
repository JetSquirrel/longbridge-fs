package broker

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	"longbridge-fs/internal/ledger"
)

// RunTWAP executes a TWAP (Time-Weighted Average Price) algo order.
// It splits the total quantity into algo_slices equal parts and submits
// one slice every (algo_duration / algo_slices) interval.
func RunTWAP(ctx context.Context, task *AlgoTask) {
	o := task.Order

	slices := o.AlgoSlices
	if slices <= 0 {
		slices = 1
	}

	dur, err := time.ParseDuration(o.AlgoDuration)
	if err != nil || dur <= 0 {
		dur = 30 * time.Minute
	}

	totalQty, err := strconv.ParseInt(o.Qty, 10, 64)
	if err != nil {
		log.Printf("TWAP %s: invalid qty %q: %v", o.IntentID, o.Qty, err)
		return
	}

	sliceQty := totalQty / int64(slices)
	if sliceQty == 0 {
		// More slices than shares: cap slices so each gets at least 1 share.
		// Use math.MaxInt32 as the upper bound to ensure safety on all platforms.
		if totalQty <= 0 {
			slices = 1
		} else if totalQty <= math.MaxInt32 {
			slices = int(totalQty)
		} else {
			slices = math.MaxInt32
		}
		sliceQty = 1
	}
	interval := dur / time.Duration(slices)
	sym := ledger.FullSymbol(o.Symbol, o.Market)

	log.Printf("TWAP %s: starting %d slices of qty=%d over %s (interval=%s)",
		o.IntentID, slices, sliceQty, dur, interval)

	for i := 1; i <= slices; i++ {
		select {
		case <-ctx.Done():
			log.Printf("TWAP %s: cancelled at slice %d/%d", o.IntentID, i, slices)
			return
		default:
		}

		// Last slice absorbs any rounding remainder.
		qty := sliceQty
		if i == slices {
			qty = totalQty - sliceQty*int64(slices-1)
		}

		sliceLabel := fmt.Sprintf("%d/%d", i, slices)
		var orderID, price string

		if task.UseMock {
			orderID = fmt.Sprintf("LOCAL-TWAP-%d-%d", time.Now().UnixNano(), i)
			price = o.Price
			if price == "" {
				price = "100.00"
			}
		} else if task.TC != nil {
			sliceOrder := o
			sliceOrder.Qty = strconv.FormatInt(qty, 10)
			var execErr error
			orderID, execErr = ExecuteOrder(ctx, task.TC, sliceOrder)
			if execErr != nil {
				log.Printf("TWAP %s slice %s failed: %v", o.IntentID, sliceLabel, execErr)
				AppendRejection(task.BcPath, o.IntentID, sym, o.Side, sliceOrder.Qty,
					fmt.Sprintf("TWAP slice %s: %v", sliceLabel, execErr))
			} else {
				price = o.Price
				if price == "" {
					price = "0.00"
				}
			}
		} else {
			return
		}

		if orderID != "" {
			AppendExecutionWithMeta(task.BcPath, o.IntentID, orderID, sym, o.Side,
				price, strconv.FormatInt(qty, 10), sliceLabel)
			log.Printf("TWAP %s: slice %s submitted order=%s qty=%d", o.IntentID, sliceLabel, orderID, qty)
		}

		// Wait for the next interval before submitting the next slice.
		if i < slices {
			select {
			case <-ctx.Done():
				log.Printf("TWAP %s: cancelled while waiting for next slice", o.IntentID)
				return
			case <-time.After(interval):
			}
		}
	}

	log.Printf("TWAP %s: completed all %d slices", o.IntentID, slices)
}
