package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"longbridge-fs/internal/credential"

	"github.com/longbridge/openapi-go/config"
	"github.com/longbridge/openapi-go/content"
	"github.com/longbridge/openapi-go/quote"
	"github.com/longbridge/openapi-go/trade"
	"github.com/spf13/cobra"
)

var (
	credentialFile string
	regionDetected string
)

func loginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Longbridge API",
		Long: `Authenticate with Longbridge API using OAuth 2.0.

Note: Currently requires manual credential setup in the credential file.
OAuth browser flow is a future enhancement.

Examples:
  longbridge-fs login
  longbridge-fs login --credential ./configs/credential`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin()
		},
	}

	return cmd
}

func runLogin() error {
	// Try to load credentials
	cfg, err := credential.Load(credentialFile)
	if err != nil {
		return fmt.Errorf(`failed to load credentials: %w

Please create a credential file at %s with the following format:

api_key=YOUR_APP_KEY
secret=YOUR_APP_SECRET
access_token=YOUR_ACCESS_TOKEN

You can obtain these credentials from the Longbridge Developer Portal.`, err, credentialFile)
	}

	// Test the credentials by creating a context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tc, err := trade.NewFromCfg(cfg)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Test by fetching account balance
	_, err = tc.AccountBalance(ctx, &trade.GetAccountBalance{})
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	fmt.Println("✓ Authentication successful")
	fmt.Printf("Credential file: %s\n", credentialFile)
	fmt.Println("\nYou can now use all Longbridge CLI commands.")

	return nil
}

func logoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Clear saved credentials",
		Long: `Clear saved authentication credentials.

Examples:
  longbridge-fs logout`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogout()
		},
	}

	return cmd
}

func runLogout() error {
	// Check if credential file exists
	if _, err := os.Stat(credentialFile); os.IsNotExist(err) {
		fmt.Println("No credentials found.")
		return nil
	}

	// Ask for confirmation
	fmt.Printf("This will remove credentials from: %s\n", credentialFile)
	fmt.Print("Are you sure? (yes/no): ")

	var response string
	fmt.Scanln(&response)

	if response != "yes" && response != "y" {
		fmt.Println("Logout cancelled.")
		return nil
	}

	// Remove credential file
	err := os.Remove(credentialFile)
	if err != nil {
		return fmt.Errorf("failed to remove credentials: %w", err)
	}

	fmt.Println("✓ Credentials cleared successfully")

	return nil
}

func checkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Verify token, region, and API connectivity",
		Long: `Verify authentication token, detect region, and test API endpoint connectivity.

Examples:
  longbridge-fs check
  longbridge-fs check --credential ./configs/credential`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheck()
		},
	}

	return cmd
}

func runCheck() error {
	fmt.Println("Checking Longbridge API connectivity...")

	// 1. Check credential file
	fmt.Printf("Credential file: %s\n", credentialFile)
	if _, err := os.Stat(credentialFile); os.IsNotExist(err) {
		fmt.Println("Status: ❌ Not found")
		return fmt.Errorf("credential file not found")
	}
	fmt.Println("Status: ✓ Found")

	// 2. Load credentials
	fmt.Print("\nLoading credentials... ")
	cfg, err := credential.Load(credentialFile)
	if err != nil {
		fmt.Println("❌ Failed")
		return fmt.Errorf("failed to load credentials: %w", err)
	}
	fmt.Println("✓ Success")

	// 3. Detect region
	fmt.Print("Detecting region... ")
	region := detectRegion()
	fmt.Printf("✓ %s\n", region)

	// 4. Test Trade API
	fmt.Print("Testing Trade API... ")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tc, err := trade.NewFromCfg(cfg)
	if err != nil {
		fmt.Println("❌ Failed")
		return fmt.Errorf("trade API connection failed: %w", err)
	}

	_, err = tc.AccountBalance(ctx, &trade.GetAccountBalance{})
	if err != nil {
		fmt.Println("❌ Failed")
		return fmt.Errorf("trade API test failed: %w", err)
	}
	fmt.Println("✓ Connected")

	// 5. Test Quote API
	fmt.Print("Testing Quote API... ")
	qc, err := quote.NewFromCfg(cfg)
	if err != nil {
		fmt.Println("❌ Failed")
		return fmt.Errorf("quote API connection failed: %w", err)
	}

	_, err = qc.Quote(ctx, []string{"AAPL.US"})
	if err != nil {
		fmt.Println("❌ Failed")
		return fmt.Errorf("quote API test failed: %w", err)
	}
	fmt.Println("✓ Connected")

	fmt.Println("\n✓ All checks passed. API is ready to use.")

	return nil
}

// detectRegion detects if the user is in China Mainland
func detectRegion() string {
	// This is a simplified implementation
	// In production, you would probe geotest.lbkrs.com and cache the result
	// For now, we'll default to "Global"
	return "Global (CN auto-detection not yet implemented)"
}

// Helper functions to create contexts

func createQuoteContext() (*quote.QuoteContext, error) {
	cfg, err := credential.Load(credentialFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	qc, err := quote.NewFromCfg(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create quote context: %w", err)
	}

	return qc, nil
}

func createTradeContext() (*trade.TradeContext, error) {
	cfg, err := credential.Load(credentialFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	tc, err := trade.NewFromCfg(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create trade context: %w", err)
	}

	return tc, nil
}

func createContentContext() (*content.ContentContext, error) {
	cfg, err := credential.Load(credentialFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	cc, err := content.NewFromCfg(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create content context: %w", err)
	}

	return cc, nil
}

func createConfigFromCredentials(credPath string) (*config.Config, error) {
	absPath, err := filepath.Abs(credPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve credential path: %w", err)
	}

	cfg, err := credential.Load(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials from %s: %w", absPath, err)
	}

	return cfg, nil
}

func createHTTPClient() (*config.Config, error) {
	cfg, err := credential.Load(credentialFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	return cfg, nil
}
