package main

import (
	"context"
	"fmt"

	"github.com/longbridge/openapi-go/http"
	"github.com/spf13/cobra"
)

func socketCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "socket",
		Short: "Socket Feed operations (OTP for WebSocket)",
		Long:  `Get Socket OTP (One-Time Password) for WebSocket authentication.`,
	}

	cmd.AddCommand(otpCmd())

	return cmd
}

func otpCmd() *cobra.Command {
	var v2 bool

	cmd := &cobra.Command{
		Use:   "otp",
		Short: "Get Socket OTP (One-Time Password)",
		Long: `Get Socket OTP for WebSocket authentication.

The OTP is required to authenticate WebSocket connections for real-time
market data and trading data feeds.

Examples:
  longbridge-fs socket otp
  longbridge-fs socket otp --v2
  longbridge-fs socket otp --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOTP(v2)
		},
	}

	cmd.Flags().BoolVar(&v2, "v2", false, "Use v2 API endpoint")

	return cmd
}

func runOTP(useV2 bool) error {
	ctx := context.Background()

	// Load credentials and create HTTP client
	cfg, err := createHTTPClient()
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}

	// Create HTTP client using the SDK's http package
	httpClient, err := http.NewFromCfg(cfg)
	if err != nil {
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Get OTP using the appropriate API version
	var otp string
	if useV2 {
		otp, err = httpClient.GetOTPV2(ctx)
	} else {
		otp, err = httpClient.GetOTP(ctx)
	}

	if err != nil {
		return fmt.Errorf("failed to get OTP: %w", err)
	}

	// Output based on format
	switch outputFormat {
	case "json":
		result := map[string]string{"otp": otp}
		return outputJSON(result)
	default:
		fmt.Printf("Socket OTP: %s\n", otp)
		fmt.Println("\nUse this OTP to authenticate your WebSocket connection.")
		fmt.Println("Note: The OTP is valid for one session only and expires after authentication.")
		return nil
	}
}
