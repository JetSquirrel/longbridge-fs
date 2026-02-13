package market

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"longbridge-fs/internal/model"

	"github.com/longportapp/openapi-go/quote"
)

// SubscriptionManager manages WebSocket-based quote subscriptions
type SubscriptionManager struct {
	qc              *quote.QuoteContext
	root            string
	subscriptions   map[string]bool // symbol -> subscribed
	mu              sync.RWMutex
	subscribeDir    string
	unsubscribeDir  string
}

// NewSubscriptionManager creates a new subscription manager
func NewSubscriptionManager(qc *quote.QuoteContext, root string) *SubscriptionManager {
	sm := &SubscriptionManager{
		qc:             qc,
		root:           root,
		subscriptions:  make(map[string]bool),
		subscribeDir:   filepath.Join(root, "quote", "subscribe"),
		unsubscribeDir: filepath.Join(root, "quote", "unsubscribe"),
	}

	// Set up quote push callback
	if qc != nil {
		qc.OnQuote(sm.handleQuotePush)
	}

	return sm
}

// ProcessSubscriptions scans for new subscribe/unsubscribe requests
func (sm *SubscriptionManager) ProcessSubscriptions(ctx context.Context) error {
	if sm.qc == nil {
		return nil // Skip if no quote context (mock mode)
	}

	// Process subscribe requests
	if err := sm.processSubscribeRequests(ctx); err != nil {
		return fmt.Errorf("process subscribe: %w", err)
	}

	// Process unsubscribe requests
	if err := sm.processUnsubscribeRequests(ctx); err != nil {
		return fmt.Errorf("process unsubscribe: %w", err)
	}

	return nil
}

// processSubscribeRequests handles new subscription requests
func (sm *SubscriptionManager) processSubscribeRequests(ctx context.Context) error {
	entries, err := os.ReadDir(sm.subscribeDir)
	if err != nil {
		return nil // Directory might not exist yet
	}

	var toSubscribe []string
	var filesToRemove []string

	sm.mu.RLock()
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		symbol := e.Name()
		if !sm.subscriptions[symbol] {
			toSubscribe = append(toSubscribe, symbol)
			filesToRemove = append(filesToRemove, filepath.Join(sm.subscribeDir, symbol))
		} else {
			// Already subscribed, just remove the file
			filesToRemove = append(filesToRemove, filepath.Join(sm.subscribeDir, symbol))
		}
	}
	sm.mu.RUnlock()

	if len(toSubscribe) == 0 {
		// Clean up any leftover files
		for _, f := range filesToRemove {
			os.Remove(f)
		}
		return nil
	}

	// Subscribe to new symbols
	err = sm.qc.Subscribe(ctx, toSubscribe, []quote.SubType{quote.SubTypeQuote}, true)
	if err != nil {
		log.Printf("subscribe failed for %v: %v", toSubscribe, err)
		return err
	}

	sm.mu.Lock()
	for _, symbol := range toSubscribe {
		sm.subscriptions[symbol] = true
	}
	sm.mu.Unlock()

	log.Printf("subscribed to real-time quotes: %v", toSubscribe)

	// Remove subscribe request files
	for _, f := range filesToRemove {
		os.Remove(f)
	}

	return nil
}

// processUnsubscribeRequests handles unsubscription requests
func (sm *SubscriptionManager) processUnsubscribeRequests(ctx context.Context) error {
	entries, err := os.ReadDir(sm.unsubscribeDir)
	if err != nil {
		return nil // Directory might not exist yet
	}

	var toUnsubscribe []string
	var filesToRemove []string

	sm.mu.RLock()
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		symbol := e.Name()
		if sm.subscriptions[symbol] {
			toUnsubscribe = append(toUnsubscribe, symbol)
		}
		filesToRemove = append(filesToRemove, filepath.Join(sm.unsubscribeDir, symbol))
	}
	sm.mu.RUnlock()

	if len(toUnsubscribe) == 0 {
		// Clean up any leftover files
		for _, f := range filesToRemove {
			os.Remove(f)
		}
		return nil
	}

	// Unsubscribe from symbols
	err = sm.qc.Unsubscribe(ctx, false, toUnsubscribe, []quote.SubType{quote.SubTypeQuote})
	if err != nil {
		log.Printf("unsubscribe failed for %v: %v", toUnsubscribe, err)
		return err
	}

	sm.mu.Lock()
	for _, symbol := range toUnsubscribe {
		delete(sm.subscriptions, symbol)
	}
	sm.mu.Unlock()

	log.Printf("unsubscribed from real-time quotes: %v", toUnsubscribe)

	// Remove unsubscribe request files
	for _, f := range filesToRemove {
		os.Remove(f)
	}

	return nil
}

// handleQuotePush is called when a real-time quote update is received
func (sm *SubscriptionManager) handleQuotePush(push *quote.PushQuote) {
	symbol := push.Symbol
	holdSymbolDir := filepath.Join(sm.root, "quote", "hold", symbol)

	if err := os.MkdirAll(holdSymbolDir, 0755); err != nil {
		log.Printf("quote push mkdir %s: %v", symbol, err)
		return
	}

	// Update overview.json with real-time data
	if err := sm.writeRealtimeOverview(holdSymbolDir, push); err != nil {
		log.Printf("quote push write %s: %v", symbol, err)
		return
	}

	log.Printf("quote push updated: %s -> hold/%s/overview.json", symbol, symbol)
}

// writeRealtimeOverview writes real-time quote data to overview.json
func (sm *SubscriptionManager) writeRealtimeOverview(dir string, push *quote.PushQuote) error {
	last := decFloat(push.LastDone)

	// Try to read previous overview to get prev_close for calculating change
	prevOverview := ReadOverview(dir)
	prev := 0.0
	if prevOverview != nil {
		prev = prevOverview.PrevClose
	}

	change := 0.0
	changePct := 0.0
	if prev != 0 {
		change = last - prev
		changePct = change / prev * 100
	}

	ov := model.QuoteOverview{
		Symbol:    push.Symbol,
		Last:      last,
		Open:      decFloat(push.Open),
		High:      decFloat(push.High),
		Low:       decFloat(push.Low),
		PrevClose: prev,
		Volume:    push.Volume,
		Turnover:  decFloat(push.Turnover),
		Change:    roundN(change, 4),
		ChangePct: roundN(changePct, 2),
		UpdatedAt: time.Unix(push.Timestamp, 0).UTC().Format(time.RFC3339),
	}

	// Also write text format for human readability
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Symbol: %s\n", push.Symbol))
	sb.WriteString(fmt.Sprintf("Last:   %s\n", decStr(push.LastDone)))
	sb.WriteString(fmt.Sprintf("Open:   %s\n", decStr(push.Open)))
	sb.WriteString(fmt.Sprintf("High:   %s\n", decStr(push.High)))
	sb.WriteString(fmt.Sprintf("Low:    %s\n", decStr(push.Low)))
	if prev != 0 {
		sb.WriteString(fmt.Sprintf("Prev:   %.2f\n", prev))
	}
	sb.WriteString(fmt.Sprintf("Volume: %d\n", push.Volume))
	sb.WriteString(fmt.Sprintf("Turnover: %s\n", decStr(push.Turnover)))
	sb.WriteString(fmt.Sprintf("Updated: %s\n", time.Unix(push.Timestamp, 0).UTC().Format(time.RFC3339)))

	os.WriteFile(filepath.Join(dir, "overview.txt"), []byte(sb.String()), 0644)

	return writeJSON(filepath.Join(dir, "overview.json"), ov)
}

// GetSubscriptions returns the list of currently subscribed symbols
func (sm *SubscriptionManager) GetSubscriptions() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	symbols := make([]string, 0, len(sm.subscriptions))
	for symbol := range sm.subscriptions {
		symbols = append(symbols, symbol)
	}
	return symbols
}

// Close cleans up the subscription manager
func (sm *SubscriptionManager) Close() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Clear subscriptions map
	sm.subscriptions = make(map[string]bool)
}
