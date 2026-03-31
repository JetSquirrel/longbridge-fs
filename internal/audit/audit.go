package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CycleLog represents one controller cycle audit log
type CycleLog struct {
	CycleID     string      `json:"cycle_id"`
	Timestamp   string      `json:"timestamp"`
	DurationMs  int64       `json:"duration_ms"`
	Steps       StepsLog    `json:"steps"`
}

// StepsLog tracks what happened in each layer during the cycle
type StepsLog struct {
	Research  ResearchStep  `json:"research,omitempty"`
	Signal    SignalStep    `json:"signal,omitempty"`
	Portfolio PortfolioStep `json:"portfolio,omitempty"`
	Risk      RiskStep      `json:"risk"`
	Execution ExecutionStep `json:"execution"`
}

type ResearchStep struct {
	FeedsRefreshed []string `json:"feeds_refreshed,omitempty"`
	Errors         []string `json:"errors,omitempty"`
}

type SignalStep struct {
	Computed []SignalComputed `json:"computed,omitempty"`
}

type SignalComputed struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
	Value  string `json:"value"`
}

type PortfolioStep struct {
	DiffComputed     bool `json:"diff_computed"`
	RebalancePending bool `json:"rebalance_pending"`
}

type RiskStep struct {
	OrdersChecked  int          `json:"orders_checked"`
	OrdersPassed   int          `json:"orders_passed"`
	OrdersRejected int          `json:"orders_rejected"`
	Rejections     []Rejection  `json:"rejections,omitempty"`
}

type Rejection struct {
	IntentID string `json:"intent_id"`
	Rule     string `json:"rule"`
}

type ExecutionStep struct {
	OrdersSubmitted int `json:"orders_submitted"`
	Executions      int `json:"executions"`
	Rejections      int `json:"rejections"`
	AlgoTasksActive int `json:"algo_tasks_active"`
}

// Logger manages audit logging
type Logger struct {
	root      string
	cycleID   string
	startTime time.Time
	log       CycleLog
}

// NewLogger creates a new audit logger for this cycle
func NewLogger(root string) *Logger {
	now := time.Now().UTC()
	cycleID := fmt.Sprintf("cycle-%s", now.Format("20060102-150405"))

	return &Logger{
		root:      root,
		cycleID:   cycleID,
		startTime: now,
		log: CycleLog{
			CycleID:   cycleID,
			Timestamp: now.Format(time.RFC3339),
			Steps:     StepsLog{},
		},
	}
}

// SetRiskStep sets the risk step data
func (l *Logger) SetRiskStep(checked, passed, rejected int, rejections []Rejection) {
	l.log.Steps.Risk = RiskStep{
		OrdersChecked:  checked,
		OrdersPassed:   passed,
		OrdersRejected: rejected,
		Rejections:     rejections,
	}
}

// SetExecutionStep sets the execution step data
func (l *Logger) SetExecutionStep(submitted, executions, rejections, algoTasksActive int) {
	l.log.Steps.Execution = ExecutionStep{
		OrdersSubmitted: submitted,
		Executions:      executions,
		Rejections:      rejections,
		AlgoTasksActive: algoTasksActive,
	}
}

// Write writes the audit log to disk
func (l *Logger) Write() error {
	// Calculate duration
	l.log.DurationMs = time.Since(l.startTime).Milliseconds()

	// Create date directory
	date := time.Now().UTC().Format("2006-01-02")
	auditDir := filepath.Join(l.root, "audit", date)
	if err := os.MkdirAll(auditDir, 0755); err != nil {
		return fmt.Errorf("failed to create audit directory: %w", err)
	}

	// Write audit log
	auditPath := filepath.Join(auditDir, l.cycleID+".json")
	data, err := json.MarshalIndent(l.log, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal audit log: %w", err)
	}

	if err := os.WriteFile(auditPath, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("failed to write audit log: %w", err)
	}

	return nil
}
