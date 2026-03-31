package broker

import (
	"context"
	"log"
	"time"
)

// executeTWAP executes a Time-Weighted Average Price algorithm
// Splits the order into equal slices and submits them at regular intervals
func (s *AlgoScheduler) executeTWAP(ctx context.Context, task *AlgoTask) {
	defer func() {
		task.mu.Lock()
		task.Done = true
		task.mu.Unlock()
	}()

	log.Printf("Starting TWAP execution: intent=%s total_qty=%d slices=%d interval=%s",
		task.IntentID, task.TotalQty, task.TotalSlices, task.Interval)

	for i := 1; i <= task.TotalSlices; i++ {
		select {
		case <-ctx.Done():
			log.Printf("TWAP execution cancelled: intent=%s at slice %d/%d", task.IntentID, i, task.TotalSlices)
			return
		default:
		}

		// Calculate quantity for this slice
		var sliceQty int64
		if i == task.TotalSlices {
			// Last slice gets remainder to handle rounding
			task.mu.Lock()
			executedQty := task.SliceQty * int64(i-1)
			sliceQty = task.TotalQty - executedQty
			task.mu.Unlock()
		} else {
			sliceQty = task.SliceQty
		}

		// Execute the slice
		if err := s.executeSlice(task, i, sliceQty); err != nil {
			log.Printf("TWAP slice %d/%d failed: intent=%s err=%v", i, task.TotalSlices, task.IntentID, err)
			// Continue with remaining slices even if one fails
		}

		task.mu.Lock()
		task.CurrentSlice = i
		task.LastSliceAt = time.Now()
		task.mu.Unlock()

		// Wait for interval before next slice (except after last slice)
		if i < task.TotalSlices {
			select {
			case <-ctx.Done():
				log.Printf("TWAP execution cancelled: intent=%s after slice %d/%d", task.IntentID, i, task.TotalSlices)
				return
			case <-time.After(task.Interval):
				// Continue to next slice
			}
		}
	}

	log.Printf("TWAP execution completed: intent=%s total_slices=%d", task.IntentID, task.TotalSlices)
}
