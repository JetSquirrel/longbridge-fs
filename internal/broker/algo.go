package broker

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"longbridge-fs/internal/ledger"
	"longbridge-fs/internal/model"

	"github.com/longbridge/openapi-go/trade"
)

// AlgoTask represents an active algorithmic order execution task
type AlgoTask struct {
	IntentID     string
	Order        model.ParsedOrder
	TotalQty     int64
	SliceQty     int64
	TotalSlices  int
	CurrentSlice int
	Interval     time.Duration
	CreatedAt    time.Time
	LastSliceAt  time.Time
	Done         bool
	Cancel       context.CancelFunc
	mu           sync.Mutex
}

// AlgoScheduler manages active algorithmic order execution tasks
type AlgoScheduler struct {
	tasks      map[string]*AlgoTask
	mu         sync.RWMutex
	bcPath     string
	tc         *trade.TradeContext
	useMock    bool
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// NewAlgoScheduler creates a new algorithm scheduler
func NewAlgoScheduler(bcPath string, tc *trade.TradeContext, useMock bool) *AlgoScheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &AlgoScheduler{
		tasks:      make(map[string]*AlgoTask),
		bcPath:     bcPath,
		tc:         tc,
		useMock:    useMock,
		ctx:        ctx,
		cancelFunc: cancel,
	}
}

// CreateTask creates and starts an algorithmic order task
func (s *AlgoScheduler) CreateTask(o model.ParsedOrder) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if task already exists
	if _, exists := s.tasks[o.IntentID]; exists {
		return fmt.Errorf("algo task already exists for intent_id: %s", o.IntentID)
	}

	// Parse total quantity
	totalQty, err := strconv.ParseInt(o.Qty, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid qty %q: %w", o.Qty, err)
	}

	// Validate algo parameters
	if o.AlgoSlices <= 0 {
		return fmt.Errorf("algo_slices must be > 0, got %d", o.AlgoSlices)
	}

	// Calculate slice quantity
	sliceQty := totalQty / int64(o.AlgoSlices)
	if sliceQty <= 0 {
		return fmt.Errorf("slice qty too small: total=%d slices=%d", totalQty, o.AlgoSlices)
	}

	// Parse duration for TWAP
	var interval time.Duration
	if o.Algo == "TWAP" {
		if o.AlgoDuration == "" {
			return fmt.Errorf("TWAP requires algo_duration")
		}
		interval, err = time.ParseDuration(o.AlgoDuration)
		if err != nil {
			return fmt.Errorf("invalid algo_duration %q: %w", o.AlgoDuration, err)
		}
		// Calculate interval between slices
		interval = interval / time.Duration(o.AlgoSlices)
	}

	// Create task context
	taskCtx, taskCancel := context.WithCancel(s.ctx)

	task := &AlgoTask{
		IntentID:     o.IntentID,
		Order:        o,
		TotalQty:     totalQty,
		SliceQty:     sliceQty,
		TotalSlices:  o.AlgoSlices,
		CurrentSlice: 0,
		Interval:     interval,
		CreatedAt:    time.Now(),
		Done:         false,
		Cancel:       taskCancel,
	}

	s.tasks[o.IntentID] = task

	// Start execution based on algo type
	switch o.Algo {
	case "TWAP":
		go s.executeTWAP(taskCtx, task)
	case "ICEBERG":
		go s.executeICEBERG(taskCtx, task)
	default:
		taskCancel()
		delete(s.tasks, o.IntentID)
		return fmt.Errorf("unsupported algo type: %s", o.Algo)
	}

	log.Printf("Created %s task for intent=%s: %d slices of %d shares", o.Algo, o.IntentID, o.AlgoSlices, sliceQty)
	return nil
}

// GetActiveCount returns the number of active algo tasks
func (s *AlgoScheduler) GetActiveCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, task := range s.tasks {
		if !task.Done {
			count++
		}
	}
	return count
}

// CleanupCompleted removes completed tasks from the scheduler
func (s *AlgoScheduler) CleanupCompleted() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for intentID, task := range s.tasks {
		if task.Done {
			delete(s.tasks, intentID)
		}
	}
}

// Shutdown gracefully stops all active tasks
func (s *AlgoScheduler) Shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Shutting down AlgoScheduler with %d active tasks", len(s.tasks))
	s.cancelFunc()

	// Wait a bit for goroutines to finish
	time.Sleep(100 * time.Millisecond)
}

// executeSlice submits a single slice of an algorithmic order
func (s *AlgoScheduler) executeSlice(task *AlgoTask, sliceNum int, qty int64) error {
	task.mu.Lock()
	order := task.Order
	intentID := task.IntentID
	task.mu.Unlock()

	sym := ledger.FullSymbol(order.Symbol, order.Market)
	sliceLabel := fmt.Sprintf("%d/%d", sliceNum, task.TotalSlices)

	var orderID string
	var price string
	var err error

	if s.useMock {
		orderID, price = ExecuteOrderMock(order)
	} else if s.tc != nil {
		// Create slice order with adjusted quantity
		sliceOrder := order
		sliceOrder.Qty = strconv.FormatInt(qty, 10)
		orderID, err = ExecuteOrder(context.Background(), s.tc, sliceOrder)
		if err != nil {
			log.Printf("Algo slice execution failed: intent=%s slice=%s err=%v", intentID, sliceLabel, err)
			return err
		}
		price = order.Price
	}

	// Append execution with slice metadata
	AppendSliceExecution(s.bcPath, intentID, orderID, sym, order.Side, price, strconv.FormatInt(qty, 10), sliceLabel, order.Algo)
	log.Printf("Algo slice executed: intent=%s slice=%s order_id=%s", intentID, sliceLabel, orderID)

	return nil
}

// AppendSliceExecution appends an EXECUTION entry with slice metadata
func AppendSliceExecution(bcPath, intentID, orderID, symbol, side, price, qty, slice, algo string) {
	AppendExecutionWithMeta(bcPath, intentID, orderID, symbol, side, price, qty, map[string]string{
		"slice": slice,
		"algo":  algo,
	})
}
