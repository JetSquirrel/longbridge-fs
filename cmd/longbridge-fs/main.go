package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"longbridge-fs/internal/account"
	"longbridge-fs/internal/broker"
	"longbridge-fs/internal/credential"
	"longbridge-fs/internal/ledger"
	"longbridge-fs/internal/market"
	"longbridge-fs/internal/risk"

	"github.com/longportapp/openapi-go/quote"
	"github.com/longportapp/openapi-go/trade"
)

// Set by ldflags at build time
var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	log.SetFlags(log.Ltime)

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		cmdInit()
	case "controller":
		cmdController()
	case "version":
		fmt.Printf("longbridge-fs %s (built %s)\n", Version, BuildTime)
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: %s <command> [options]

Commands:
  init         Initialize the FS directory structure
  controller   Start the trade controller daemon
  version      Print version information

Options for init:
  --root PATH                 FS root directory (default: .)

Options for controller:
  --root PATH                 FS root directory (default: .)
  --interval DURATION         Poll interval (default: 2s)
  --credential FILE           Credential file path (default: credential)
  --mock                      Use mock execution without API (default: false)
  --compact-after N           Compact after N executed orders, 0=disable (default: 10)
`, os.Args[0])
}

// cmdInit creates the FS directory structure.
func cmdInit() {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	root := fs.String("root", ".", "FS root directory")
	fs.Parse(os.Args[2:])

	dirs := []string{
		filepath.Join(*root, "account"),
		filepath.Join(*root, "trade", "blocks"),
		filepath.Join(*root, "quote", "hold"),
		filepath.Join(*root, "quote", "track"),
		filepath.Join(*root, "quote", "subscribe"),
		filepath.Join(*root, "quote", "unsubscribe"),
		filepath.Join(*root, "quote", "market"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			log.Fatalf("mkdir %s: %v", d, err)
		}
	}

	// Default beancount ledger
	bcPath := filepath.Join(*root, "trade", "beancount.txt")
	if _, err := os.Stat(bcPath); os.IsNotExist(err) {
		os.WriteFile(bcPath, []byte("; beancount append-only trade ledger\n"), 0644)
	}

	// Default account state
	statePath := filepath.Join(*root, "account", "state.json")
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		os.WriteFile(statePath, []byte(`{"updated_at":"","cash":[],"positions":[],"orders":[]}`+"\n"), 0644)
	}

	// Default risk control config
	rcPath := filepath.Join(*root, "trade", "risk_control.json")
	if _, err := os.Stat(rcPath); os.IsNotExist(err) {
		os.WriteFile(rcPath, []byte("{}\n"), 0644)
	}

	log.Printf("initialized FS at %s", *root)
}

// cmdController starts the trade controller daemon.
func cmdController() {
	fs := flag.NewFlagSet("controller", flag.ExitOnError)
	root := fs.String("root", ".", "FS root directory")
	interval := fs.Duration("interval", 2*time.Second, "Poll interval")
	credFile := fs.String("credential", "credential", "Credential file path")
	mock := fs.Bool("mock", false, "Use mock execution without API")
	compactAfter := fs.Int("compact-after", 10, "Compact after N executed orders, 0=disable")
	fs.Parse(os.Args[2:])

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("received signal, shutting down...")
		cancel()
	}()

	var tc *trade.TradeContext
	var qc *quote.QuoteContext
	var subManager *market.SubscriptionManager
	useMock := *mock

	if !useMock {
		cfg, err := credential.Load(*credFile)
		if err != nil {
			log.Printf("credential load failed: %v (falling back to mock mode)", err)
			useMock = true
		} else {
			// Initialize trade context
			tctx, err := trade.NewFromCfg(cfg)
			if err != nil {
				log.Printf("trade context init failed: %v (falling back to mock mode)", err)
				useMock = true
			} else {
				tc = tctx
			}

			// Initialize quote context
			qctx, err := quote.NewFromCfg(cfg)
			if err != nil {
				log.Printf("quote context init failed: %v (quote disabled)", err)
			} else {
				qc = qctx
			}

			if tc != nil {
				log.Println("connected to Longbridge API")
			}
		}
	}

	if useMock {
		log.Println("running in MOCK mode (no API calls)")
	}

	// Initialize subscription manager
	subManager = market.NewSubscriptionManager(qc, *root)
	if qc != nil {
		log.Println("WebSocket subscription manager initialized")
	}

	log.Printf("controller started: root=%s interval=%s compact-after=%d", *root, *interval, *compactAfter)

	executedCount := 0
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Kill switch
			killPath := filepath.Join(*root, ".kill")
			if _, err := os.Stat(killPath); err == nil {
				log.Println("kill switch activated, shutting down...")
				os.Remove(killPath)
				cancel()
				return
			}

			// Process trade ledger
			n, err := broker.ProcessLedger(ctx, tc, *root, useMock)
			if err != nil {
				log.Printf("process failed: %v", err)
			}
			executedCount += n

			// Refresh account state (only with real API)
			if tc != nil {
				if err := account.RefreshState(ctx, tc, *root); err != nil {
					log.Printf("account refresh failed: %v", err)
				}
			}

			// Process WebSocket subscription requests (subscribe/unsubscribe)
			if subManager != nil {
				if err := subManager.ProcessSubscriptions(ctx); err != nil {
					log.Printf("subscription processing failed: %v", err)
				}
			}

			// Refresh quotes via track files (one-shot poll-based)
			if qc != nil {
				market.RefreshQuotes(ctx, qc, *root)
			}

			// Generate PnL report (positions + current prices — file-only, works in mock)
			if err := account.GeneratePnL(*root); err != nil {
				log.Printf("pnl generation failed: %v", err)
			}

			// Generate portfolio summary (all hold quotes + positions)
			if err := account.GeneratePortfolio(*root); err != nil {
				log.Printf("portfolio generation failed: %v", err)
			}

			// Risk control: stop-loss / take-profit
			if err := risk.CheckRiskRules(*root); err != nil {
				log.Printf("risk check failed: %v", err)
			}

			// Compaction
			if *compactAfter > 0 && executedCount >= *compactAfter {
				if err := ledger.CompactBlocks(*root, executedCount); err != nil {
					log.Printf("compact failed: %v", err)
				} else {
					executedCount = 0
				}
			}
		}
	}
}
