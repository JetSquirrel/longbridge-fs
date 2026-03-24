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
	"longbridge-fs/internal/risk"

	"github.com/longportapp/openapi-go/quote"
	"github.com/longportapp/openapi-go/trade"
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
		filepath.Join(root, "account"),
		filepath.Join(root, "trade", "blocks"),
		filepath.Join(root, "quote", "hold"),
		filepath.Join(root, "quote", "track"),
		filepath.Join(root, "quote", "subscribe"),
		filepath.Join(root, "quote", "unsubscribe"),
		filepath.Join(root, "quote", "market"),
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

	// Default risk control config
	rcPath := filepath.Join(root, "trade", "risk_control.json")
	if _, err := os.Stat(rcPath); os.IsNotExist(err) {
		if err := os.WriteFile(rcPath, []byte("{}\n"), 0644); err != nil {
			return fmt.Errorf("failed to create risk control config: %w", err)
		}
		if verbose {
			log.Printf("created file: %s", rcPath)
		}
	}

	log.Printf("✓ Successfully initialized FS at %s", root)
	return nil
}

// controllerCmd creates the controller subcommand
func controllerCmd() *cobra.Command {
	var (
		root         string
		interval     time.Duration
		credFile     string
		mock         bool
		compactAfter int
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
			return runController(root, interval, credFile, mock, compactAfter)
		},
	}

	cmd.Flags().StringVar(&root, "root", ".", "FS root directory")
	cmd.Flags().DurationVar(&interval, "interval", 2*time.Second, "Poll interval")
	cmd.Flags().StringVar(&credFile, "credential", "credential", "Credential file path")
	cmd.Flags().BoolVar(&mock, "mock", false, "Use mock execution without API")
	cmd.Flags().IntVar(&compactAfter, "compact-after", 10, "Compact after N executed orders, 0=disable")

	return cmd
}

func runController(root string, interval time.Duration, credFile string, mock bool, compactAfter int) error {
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
