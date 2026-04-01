package broker

import (
	"context"
	"log"
	"sync"

	"longbridge-fs/internal/model"

	"github.com/longbridge/openapi-go/trade"
)

// AlgoTask holds all state for an in-flight algorithmic order.
type AlgoTask struct {
	Order   model.ParsedOrder
	BcPath  string
	TC      *trade.TradeContext
	UseMock bool
}

// AlgoScheduler manages active algo goroutines, preventing duplicate
// scheduling for the same intent_id across controller ticks.
type AlgoScheduler struct {
	mu     sync.Mutex
	active map[string]bool // intentID -> running
}

// NewAlgoScheduler creates a new AlgoScheduler.
func NewAlgoScheduler() *AlgoScheduler {
	return &AlgoScheduler{active: make(map[string]bool)}
}

// IsActive returns true if an algo task is already running for the given intentID.
func (s *AlgoScheduler) IsActive(intentID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active[intentID]
}

// ActiveCount returns the number of currently running algo tasks.
func (s *AlgoScheduler) ActiveCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.active)
}

// Submit launches the algo task as a goroutine.
func (s *AlgoScheduler) Submit(ctx context.Context, task *AlgoTask) {
	s.mu.Lock()
	s.active[task.Order.IntentID] = true
	s.mu.Unlock()

	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.active, task.Order.IntentID)
			s.mu.Unlock()
		}()

		switch task.Order.Algo {
		case "TWAP":
			RunTWAP(ctx, task)
		case "ICEBERG":
			RunICEBERG(ctx, task)
		default:
			log.Printf("algo scheduler: unknown algo type %q for intent %s", task.Order.Algo, task.Order.IntentID)
		}
	}()
}
