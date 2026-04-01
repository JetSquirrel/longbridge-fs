package broker

import (
	"context"
	"log"
	"time"
)

// executeICEBERG executes an Iceberg algorithm
// Submits partial orders sequentially, revealing only a portion at a time
// Each slice is submitted after the previous one, simulating fill-based progression
func (s *AlgoScheduler) executeICEBERG(ctx context.Context, task *AlgoTask) {
	defer func() {
		task.mu.Lock()
		task.Done = true
		task.mu.Unlock()
	}()

	log.Printf("Starting ICEBERG execution: intent=%s total_qty=%d slices=%d visible_qty=%d",
		task.IntentID, task.TotalQty, task.TotalSlices, task.SliceQty)

	// For ICEBERG, we submit slices sequentially without time delay
	// In a real implementation, we would wait for fills, but for this implementation
	// we simulate by submitting each slice with a small delay to represent processing time
	for i := 1; i <= task.TotalSlices; i++ {
		select {
		case <-ctx.Done():
			log.Printf("ICEBERG execution cancelled: intent=%s at slice %d/%d", task.IntentID, i, task.TotalSlices)
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
			log.Printf("ICEBERG slice %d/%d failed: intent=%s err=%v", i, task.TotalSlices, task.IntentID, err)
			// Continue with remaining slices even if one fails
		}

		task.mu.Lock()
		task.CurrentSlice = i
		task.LastSliceAt = time.Now()
		task.mu.Unlock()

		// Small delay between slices to simulate fill detection and next order submission
		// In a real implementation, this would wait for order fill confirmation
		if i < task.TotalSlices {
			select {
			case <-ctx.Done():
				log.Printf("ICEBERG execution cancelled: intent=%s after slice %d/%d", task.IntentID, i, task.TotalSlices)
				return
			case <-time.After(2 * time.Second):
				// Continue to next slice
			}
		}
	}

	log.Printf("ICEBERG execution completed: intent=%s total_slices=%d", task.IntentID, task.TotalSlices)
}
