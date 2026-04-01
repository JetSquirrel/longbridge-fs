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

// RunICEBERG executes an ICEBERG algo order.
// It shows a visible portion (total_qty / algo_slices) at a time; once
// a slice is submitted the scheduler waits for the configured poll interval
// before submitting the next one, simulating an order-fill trigger.
func RunICEBERG(ctx context.Context, task *AlgoTask) {
	o := task.Order

	slices := o.AlgoSlices
	if slices <= 0 {
		slices = 1
	}

	totalQty, err := strconv.ParseInt(o.Qty, 10, 64)
	if err != nil {
		log.Printf("ICEBERG %s: invalid qty %q: %v", o.IntentID, o.Qty, err)
		return
	}

	visibleQty := totalQty / int64(slices)
	if visibleQty == 0 {
		// More slices than shares: cap slices so each gets at least 1 share.
		// Use math.MaxInt32 as the upper bound to ensure safety on all platforms.
		if totalQty <= 0 {
			slices = 1
		} else if totalQty <= math.MaxInt32 {
			slices = int(totalQty)
		} else {
			slices = math.MaxInt32
		}
		visibleQty = 1
	}

	// Use algo_duration as the wait between slices; default to 5 seconds.
	pollInterval := 5 * time.Second
	if o.AlgoDuration != "" {
		if dur, err := time.ParseDuration(o.AlgoDuration); err == nil && dur > 0 {
			pollInterval = dur / time.Duration(slices)
		}
	}

	sym := ledger.FullSymbol(o.Symbol, o.Market)

	log.Printf("ICEBERG %s: starting %d slices of visible_qty=%d (poll=%s)",
		o.IntentID, slices, visibleQty, pollInterval)

	for i := 1; i <= slices; i++ {
		select {
		case <-ctx.Done():
			log.Printf("ICEBERG %s: cancelled at slice %d/%d", o.IntentID, i, slices)
			return
		default:
		}

		// Last slice absorbs any rounding remainder.
		qty := visibleQty
		if i == slices {
			qty = totalQty - visibleQty*int64(slices-1)
		}

		sliceLabel := fmt.Sprintf("%d/%d", i, slices)
		var orderID, price string

		if task.UseMock {
			orderID = fmt.Sprintf("LOCAL-ICEBERG-%d-%d", time.Now().UnixNano(), i)
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
				log.Printf("ICEBERG %s slice %s failed: %v", o.IntentID, sliceLabel, execErr)
				AppendRejection(task.BcPath, o.IntentID, sym, o.Side, sliceOrder.Qty,
					fmt.Sprintf("ICEBERG slice %s: %v", sliceLabel, execErr))
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
			log.Printf("ICEBERG %s: slice %s submitted order=%s qty=%d", o.IntentID, sliceLabel, orderID, qty)
		}

		// Wait for fill simulation before next slice.
		if i < slices {
			select {
			case <-ctx.Done():
				log.Printf("ICEBERG %s: cancelled while waiting for fill on slice %s", o.IntentID, sliceLabel)
				return
			case <-time.After(pollInterval):
			}
		}
	}

	log.Printf("ICEBERG %s: completed all %d slices", o.IntentID, slices)
}
