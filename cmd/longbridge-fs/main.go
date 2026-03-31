package main

import (
	"context"
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
	"longbridge-fs/internal/portfolio"
	"longbridge-fs/internal/research"
	"longbridge-fs/internal/risk"
	signalpkg "longbridge-fs/internal/signal"

	"github.com/longbridge/openapi-go/quote"
	"github.com/longbridge/openapi-go/trade"
	"github.com/spf13/cobra"
)

// Set by ldflags at build time
var (
	Version   = "dev"
	BuildTime = "unknown"
)

// Global flags
var (
	rootDir      string
	verbose      bool
	outputFormat string
)

func main() {
	log.SetFlags(log.Ltime)

	rootCmd := &cobra.Command{
		Use:   "longbridge-fs",
		Short: "AI-native CLI for Longbridge trading platform",
		Long: `Longbridge Terminal - AI-native CLI for the Longbridge trading platform

Real-time market data, portfolio management, and trading operations.
Covers every Longbridge OpenAPI endpoint for HK/US/CN markets.

Designed for scripting, AI-agent tool-calling, and daily trading workflows.`,
		Version: fmt.Sprintf("%s (built %s)", Version, BuildTime),
	}

	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVar(&outputFormat, "format", "table", "Output format (table, json, csv)")
	rootCmd.PersistentFlags().StringVar(&credentialFile, "credential", "credential", "Credential file path")

	// Add subcommands

	// Legacy file-system based commands
	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(controllerCmd())

	// New AI-native CLI commands

	// Authentication
	rootCmd.AddCommand(loginCmd())
	rootCmd.AddCommand(logoutCmd())
	rootCmd.AddCommand(checkCmd())

	// Market data
	rootCmd.AddCommand(quoteCmd())
	rootCmd.AddCommand(staticCmd())
	rootCmd.AddCommand(depthCmd())
	rootCmd.AddCommand(klinesCmd())
	rootCmd.AddCommand(intradayCmd())
	rootCmd.AddCommand(filingsCmd())

	// Content
	rootCmd.AddCommand(contentCmd())

	// Socket Feed
	rootCmd.AddCommand(socketCmd())

	// Account & portfolio
	rootCmd.AddCommand(accountCmd())

	// Trading
	rootCmd.AddCommand(orderCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// initCmd creates the init subcommand
func initCmd() *cobra.Command {
	var root string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize the FS directory structure",
		Long: `Initialize directory structure for longbridge-fs

Creates the required directories and default configuration files
for the file system-based trading framework.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(root)
		},
	}

	cmd.Flags().StringVar(&root, "root", ".", "FS root directory")
	return cmd
}

func runInit(root string) error {
	dirs := []string{
		// Existing directories
		filepath.Join(root, "account"),
		filepath.Join(root, "trade", "blocks"),
		filepath.Join(root, "quote", "hold"),
		filepath.Join(root, "quote", "track"),
		filepath.Join(root, "quote", "subscribe"),
		filepath.Join(root, "quote", "unsubscribe"),
		filepath.Join(root, "quote", "market"),
		// Phase 1: L1 Research Layer
		filepath.Join(root, "research", "feeds", "news"),
		filepath.Join(root, "research", "feeds", "topics"),
		filepath.Join(root, "research", "feeds", "custom"),
		// Phase 1: L2 Signal Layer
		filepath.Join(root, "signal", "definitions"),
		filepath.Join(root, "signal", "output"),
		// Phase 1: L3 Portfolio Layer
		filepath.Join(root, "portfolio", "rebalance"),
		filepath.Join(root, "portfolio", "history"),
		// Phase 1: L4 Risk Control Layer (new structure)
		filepath.Join(root, "trade", "risk"),
		// Phase 1: Audit Layer
		filepath.Join(root, "audit"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", d, err)
		}
		if verbose {
			log.Printf("created directory: %s", d)
		}
	}

	// Default beancount ledger
	bcPath := filepath.Join(root, "trade", "beancount.txt")
	if _, err := os.Stat(bcPath); os.IsNotExist(err) {
		if err := os.WriteFile(bcPath, []byte("; beancount append-only trade ledger\n"), 0644); err != nil {
			return fmt.Errorf("failed to create beancount ledger: %w", err)
		}
		if verbose {
			log.Printf("created file: %s", bcPath)
		}
	}

	// Default account state
	statePath := filepath.Join(root, "account", "state.json")
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		if err := os.WriteFile(statePath, []byte(`{"updated_at":"","cash":[],"positions":[],"orders":[]}`+"\n"), 0644); err != nil {
			return fmt.Errorf("failed to create account state: %w", err)
		}
		if verbose {
			log.Printf("created file: %s", statePath)
		}
	}

	// Default risk control config (legacy, kept for backward compatibility)
	rcPath := filepath.Join(root, "trade", "risk_control.json")
	if _, err := os.Stat(rcPath); os.IsNotExist(err) {
		if err := os.WriteFile(rcPath, []byte("{}\n"), 0644); err != nil {
			return fmt.Errorf("failed to create risk control config: %w", err)
		}
		if verbose {
			log.Printf("created file: %s", rcPath)
		}
	}

	// Phase 1: L1 Research watchlist
	watchlistPath := filepath.Join(root, "research", "watchlist.json")
	if _, err := os.Stat(watchlistPath); os.IsNotExist(err) {
		watchlistDefault := `{
  "symbols": [],
  "refresh_interval": "5m",
  "feeds": ["news", "topics"]
}
`
		if err := os.WriteFile(watchlistPath, []byte(watchlistDefault), 0644); err != nil {
			return fmt.Errorf("failed to create watchlist: %w", err)
		}
		if verbose {
			log.Printf("created file: %s", watchlistPath)
		}
	}

	// Phase 1: L1 Research summary
	summaryPath := filepath.Join(root, "research", "summary.json")
	if _, err := os.Stat(summaryPath); os.IsNotExist(err) {
		summaryDefault := `{
  "updated_at": "",
  "symbols": {}
}
`
		if err := os.WriteFile(summaryPath, []byte(summaryDefault), 0644); err != nil {
			return fmt.Errorf("failed to create research summary: %w", err)
		}
		if verbose {
			log.Printf("created file: %s", summaryPath)
		}
	}

	// Phase 1: L2 Signal active signals
	activeSignalsPath := filepath.Join(root, "signal", "active.json")
	if _, err := os.Stat(activeSignalsPath); os.IsNotExist(err) {
		activeDefault := `{
  "updated_at": "",
  "signals": []
}
`
		if err := os.WriteFile(activeSignalsPath, []byte(activeDefault), 0644); err != nil {
			return fmt.Errorf("failed to create active signals: %w", err)
		}
		if verbose {
			log.Printf("created file: %s", activeSignalsPath)
		}
	}

	// Phase 1: L3 Portfolio current
	currentPortfolioPath := filepath.Join(root, "portfolio", "current.json")
	if _, err := os.Stat(currentPortfolioPath); os.IsNotExist(err) {
		currentDefault := `{
  "updated_at": "",
  "total_equity": 0.0,
  "positions": {},
  "cash": 0.0,
  "cash_pct": 0.0
}
`
		if err := os.WriteFile(currentPortfolioPath, []byte(currentDefault), 0644); err != nil {
			return fmt.Errorf("failed to create current portfolio: %w", err)
		}
		if verbose {
			log.Printf("created file: %s", currentPortfolioPath)
		}
	}

	// Phase 1: L4 Risk policy
	policyPath := filepath.Join(root, "trade", "risk", "policy.json")
	if _, err := os.Stat(policyPath); os.IsNotExist(err) {
		policyDefault := `{
  "version": 1,
  "enabled": false,
  "mode": "ENFORCE",
  "pre_trade_checks": true,
  "post_trade_monitoring": true,
  "daily_loss_limit": {
    "enabled": false,
    "max_loss_pct": 0.03,
    "action": "HALT"
  },
  "order_frequency": {
    "enabled": false,
    "max_orders_per_hour": 20,
    "max_orders_per_day": 100
  }
}
`
		if err := os.WriteFile(policyPath, []byte(policyDefault), 0644); err != nil {
			return fmt.Errorf("failed to create risk policy: %w", err)
		}
		if verbose {
			log.Printf("created file: %s", policyPath)
		}
	}

	// Phase 1: L4 Pre-trade rules
	preTradeRulesPath := filepath.Join(root, "trade", "risk", "pre_trade.json")
	if _, err := os.Stat(preTradeRulesPath); os.IsNotExist(err) {
		preTradeDefault := `{
  "max_single_order_pct": 0.10,
  "max_single_order_value": 50000,
  "allowed_symbols": [],
  "blocked_symbols": [],
  "allowed_sides": ["BUY", "SELL"],
  "require_limit_price": false,
  "max_deviation_from_market_pct": 0.05
}
`
		if err := os.WriteFile(preTradeRulesPath, []byte(preTradeDefault), 0644); err != nil {
			return fmt.Errorf("failed to create pre-trade rules: %w", err)
		}
		if verbose {
			log.Printf("created file: %s", preTradeRulesPath)
		}
	}

	// Phase 1: L4 Position limits
	positionLimitsPath := filepath.Join(root, "trade", "risk", "position_limits.json")
	if _, err := os.Stat(positionLimitsPath); os.IsNotExist(err) {
		positionLimitsDefault := `{
  "max_position_pct": 0.25,
  "max_positions_count": 15,
  "sector_limits": {},
  "per_symbol_limits": {}
}
`
		if err := os.WriteFile(positionLimitsPath, []byte(positionLimitsDefault), 0644); err != nil {
			return fmt.Errorf("failed to create position limits: %w", err)
		}
		if verbose {
			log.Printf("created file: %s", positionLimitsPath)
		}
	}

	// Phase 1: L4 Daily limits
	dailyLimitsPath := filepath.Join(root, "trade", "risk", "daily_limits.json")
	if _, err := os.Stat(dailyLimitsPath); os.IsNotExist(err) {
		dailyLimitsDefault := `{
  "date": "",
  "starting_equity": 0.0,
  "current_equity": 0.0,
  "realized_pnl": 0.0,
  "unrealized_pnl": 0.0,
  "total_pnl_pct": 0.0,
  "orders_this_hour": 0,
  "orders_today": 0,
  "is_halted": false,
  "halt_reason": null
}
`
		if err := os.WriteFile(dailyLimitsPath, []byte(dailyLimitsDefault), 0644); err != nil {
			return fmt.Errorf("failed to create daily limits: %w", err)
		}
		if verbose {
			log.Printf("created file: %s", dailyLimitsPath)
		}
	}

	// Phase 1: L4 Risk status
	statusPath := filepath.Join(root, "trade", "risk", "status.json")
	if _, err := os.Stat(statusPath); os.IsNotExist(err) {
		statusDefault := `{
  "updated_at": "",
  "checks_today": 0,
  "checks_passed": 0,
  "checks_rejected": 0,
  "is_halted": false,
  "halt_reason": null
}
`
		if err := os.WriteFile(statusPath, []byte(statusDefault), 0644); err != nil {
			return fmt.Errorf("failed to create risk status: %w", err)
		}
		if verbose {
			log.Printf("created file: %s", statusPath)
		}
	}

	log.Printf("✓ Successfully initialized FS at %s", root)
	log.Printf("✓ Phase 1: Five-layer harness directories created")
	return nil
}

// controllerCmd creates the controller subcommand
func controllerCmd() *cobra.Command {
	var (
		root          string
		interval      time.Duration
		credFile      string
		mock          bool
		compactAfter  int
		autoRebalance bool
	)

	cmd := &cobra.Command{
		Use:   "controller",
		Short: "Start the trade controller daemon",
		Long: `Start the trade controller daemon

The controller monitors the file system and automatically:
  - Processes new orders from beancount.txt
  - Refreshes account state and positions
  - Updates real-time quotes via WebSocket subscriptions
  - Generates PnL and portfolio reports
  - Enforces risk control rules (stop-loss/take-profit)
  - Compacts ledger history into blocks`,
		Example: `  # Run with real API
  longbridge-fs controller --root ./fs --credential ./configs/credential

  # Run in mock mode (no API calls)
  longbridge-fs controller --root ./fs --mock

  # Custom polling interval
  longbridge-fs controller --root ./fs --interval 5s`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runController(root, interval, credFile, mock, compactAfter, autoRebalance)
		},
	}

	cmd.Flags().StringVar(&root, "root", ".", "FS root directory")
	cmd.Flags().DurationVar(&interval, "interval", 2*time.Second, "Poll interval")
	cmd.Flags().StringVar(&credFile, "credential", "credential", "Credential file path")
	cmd.Flags().BoolVar(&mock, "mock", false, "Use mock execution without API")
	cmd.Flags().IntVar(&compactAfter, "compact-after", 10, "Compact after N executed orders, 0=disable")
	cmd.Flags().BoolVar(&autoRebalance, "auto-rebalance", false, "Automatically create rebalance orders when portfolio drift is detected")

	return cmd
}

func runController(root string, interval time.Duration, credFile string, mock bool, compactAfter int, autoRebalance bool) error {
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
	useMock := mock

	if !useMock {
		cfg, err := credential.Load(credFile)
		if err != nil {
			log.Printf("⚠ Credential load failed: %v (falling back to mock mode)", err)
			useMock = true
		} else {
			// Initialize trade context
			tctx, err := trade.NewFromCfg(cfg)
			if err != nil {
				log.Printf("⚠ Trade context init failed: %v (falling back to mock mode)", err)
				useMock = true
			} else {
				tc = tctx
			}

			// Initialize quote context
			qctx, err := quote.NewFromCfg(cfg)
			if err != nil {
				log.Printf("⚠ Quote context init failed: %v (quote disabled)", err)
			} else {
				qc = qctx
			}

			if tc != nil {
				log.Println("✓ Connected to Longbridge API")
			}
		}
	}

	if useMock {
		log.Println("🔧 Running in MOCK mode (no API calls)")
	}

	// Initialize subscription manager
	subManager = market.NewSubscriptionManager(qc, root)
	if qc != nil {
		log.Println("✓ WebSocket subscription manager initialized")
	}

	if verbose {
		log.Printf("Controller configuration:")
		log.Printf("  Root: %s", root)
		log.Printf("  Interval: %s", interval)
		log.Printf("  Compact after: %d orders", compactAfter)
		log.Printf("  Mock mode: %v", useMock)
		log.Printf("  Auto-rebalance: %v", autoRebalance)
	}

	log.Printf("🚀 Controller started (interval=%s, compact-after=%d)", interval, compactAfter)

	executedCount := 0
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("✓ Controller stopped gracefully")
			return nil
		case <-ticker.C:
			// Kill switch
			killPath := filepath.Join(root, ".kill")
			if _, err := os.Stat(killPath); err == nil {
				log.Println("🛑 Kill switch activated, shutting down...")
				os.Remove(killPath)
				cancel()
				return nil
			}

			// Process trade ledger
			n, err := broker.ProcessLedger(ctx, tc, root, useMock)
			if err != nil {
				log.Printf("❌ Order processing failed: %v", err)
			} else if n > 0 && verbose {
				log.Printf("✓ Processed %d order(s)", n)
			}
			executedCount += n

			// Refresh account state (only with real API)
			if tc != nil {
				if err := account.RefreshState(ctx, tc, root); err != nil {
					log.Printf("❌ Account refresh failed: %v", err)
				} else if verbose {
					log.Printf("✓ Account state refreshed")
				}
			}

			// Process WebSocket subscription requests (subscribe/unsubscribe)
			if subManager != nil {
				if err := subManager.ProcessSubscriptions(ctx); err != nil {
					log.Printf("❌ Subscription processing failed: %v", err)
				}
			}

			// Refresh quotes via track files (one-shot poll-based)
			if qc != nil {
				market.RefreshQuotes(ctx, qc, root)
			}

			// Phase 3: Refresh research feeds (news/topics from Content API)
			if !useMock {
				if err := research.RefreshFeeds(ctx, root, credFile); err != nil {
					// Don't fail the entire cycle for research refresh errors
					if verbose {
						log.Printf("⚠ Research refresh failed: %v", err)
					}
				} else if verbose {
					log.Printf("✓ Research feeds refreshed")
				}
			}

			// Phase 3: Compute builtin signals from signal/definitions/
			if err := signalpkg.ComputeAll(root); err != nil {
				if verbose {
					log.Printf("⚠ Signal computation failed: %v", err)
				}
			} else if verbose {
				log.Printf("✓ Signals computed")
			}

			// Generate PnL report (positions + current prices — file-only, works in mock)
			if err := account.GeneratePnL(root); err != nil {
				log.Printf("❌ PnL generation failed: %v", err)
			} else if verbose {
				log.Printf("✓ PnL report generated")
			}

			// Generate portfolio summary (all hold quotes + positions)
			if err := account.GeneratePortfolio(root); err != nil {
				log.Printf("❌ Portfolio generation failed: %v", err)
			} else if verbose {
				log.Printf("✓ Portfolio summary generated")
			}

			// Phase 2: Portfolio construction - sync current portfolio state
			if err := portfolio.SyncCurrent(root); err != nil {
				log.Printf("❌ Portfolio sync failed: %v", err)
			} else if verbose {
				log.Printf("✓ Portfolio current state synced")
			}

			// Phase 2: Compute portfolio diff (target vs current)
			if err := portfolio.ComputeDiff(root); err != nil {
				log.Printf("❌ Portfolio diff computation failed: %v", err)
			} else if verbose {
				log.Printf("✓ Portfolio diff computed")
			}

			// Phase 2: Auto-rebalance mode: create pending.json from diff when drift detected
			if autoRebalance {
				if err := portfolio.AutoCreatePending(root); err != nil {
					log.Printf("❌ Auto-rebalance failed: %v", err)
				} else if verbose {
					log.Printf("✓ Auto-rebalance check complete")
				}
			}

			// Phase 2: Process pending rebalance orders
			if err := portfolio.ProcessRebalance(root); err != nil {
				log.Printf("❌ Rebalance processing failed: %v", err)
			} else if verbose {
				log.Printf("✓ Rebalance processed")
			}

			// Risk control: stop-loss / take-profit
			if err := risk.CheckRiskRules(root); err != nil {
				log.Printf("❌ Risk check failed: %v", err)
			}

			// Compaction
			if compactAfter > 0 && executedCount >= compactAfter {
				if err := ledger.CompactBlocks(root, executedCount); err != nil {
					log.Printf("❌ Compaction failed: %v", err)
				} else {
					log.Printf("✓ Compacted %d executed orders into blocks", executedCount)
					executedCount = 0
				}
			}
		}
	}
}
