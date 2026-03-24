package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/longportapp/openapi-go/quote"
	"github.com/spf13/cobra"
)

func quoteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quote [symbols...]",
		Short: "Get real-time quotes for symbols",
		Long: `Get real-time quotes for one or more symbols.

Examples:
  longbridge-fs quote TSLA.US NVDA.US
  longbridge-fs quote AAPL.US --format json
  longbridge-fs quote 700.HK 9988.HK --format table`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("at least one symbol is required")
			}
			return runQuote(args)
		},
	}

	return cmd
}

func staticCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "static [symbols...]",
		Short: "Get static quotes in table format",
		Long: `Get static quotes displayed in a table format.

Examples:
  longbridge-fs static NVDA.US
  longbridge-fs static TSLA.US NVDA.US`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("at least one symbol is required")
			}
			// Force table format for static command
			outputFormat = "table"
			return runQuote(args)
		},
	}

	return cmd
}

func runQuote(symbols []string) error {
	ctx := context.Background()

	// Load credentials and create quote context
	qc, err := createQuoteContext()
	if err != nil {
		return fmt.Errorf("failed to initialize quote context: %w", err)
	}

	// Fetch quotes
	quotes, err := qc.Quote(ctx, symbols)
	if err != nil {
		return fmt.Errorf("failed to fetch quotes: %w", err)
	}

	// Output based on format
	switch outputFormat {
	case "json":
		return outputJSON(quotes)
	case "table":
		return outputQuotesTable(quotes)
	default:
		return outputQuotesTable(quotes)
	}
}

func outputQuotesTable(quotes []*quote.SecurityQuote) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Print header
	fmt.Fprintln(w, "| Symbol\t| Last\t| Prev Close\t| Open\t| High\t| Low\t| Volume\t| Turnover\t| Status\t|")
	fmt.Fprintln(w, "|-------\t|------\t|-----------\t|------\t|------\t|-----\t|---------\t|-----------\t|-------\t|")

	for _, q := range quotes {
		status := fmt.Sprintf("%v", q.TradeStatus)

		fmt.Fprintf(w, "| %s\t| %.3f\t| %.3f\t| %.3f\t| %.3f\t| %.3f\t| %d\t| %.3f\t| %s\t|\n",
			q.Symbol,
			decFloat(q.LastDone),
			decFloat(q.PrevClose),
			decFloat(q.Open),
			decFloat(q.High),
			decFloat(q.Low),
			q.Volume,
			decFloat(q.Turnover),
			status,
		)
	}

	return w.Flush()
}

func outputJSON(v interface{}) error {
	// Create a simplified structure for JSON output
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

// Helper function to format quotes for JSON output
func formatQuotesForJSON(quotes []*quote.SecurityQuote) []map[string]interface{} {
	result := make([]map[string]interface{}, len(quotes))
	for i, q := range quotes {
		status := fmt.Sprintf("%v", q.TradeStatus)

		result[i] = map[string]interface{}{
			"symbol":     q.Symbol,
			"last":       fmt.Sprintf("%.3f", decFloat(q.LastDone)),
			"prev_close": fmt.Sprintf("%.3f", decFloat(q.PrevClose)),
			"open":       fmt.Sprintf("%.3f", decFloat(q.Open)),
			"high":       fmt.Sprintf("%.3f", decFloat(q.High)),
			"low":        fmt.Sprintf("%.3f", decFloat(q.Low)),
			"volume":     fmt.Sprintf("%d", q.Volume),
			"turnover":   fmt.Sprintf("%.3f", decFloat(q.Turnover)),
			"status":     status,
		}
	}
	return result
}

func depthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "depth [symbol]",
		Short: "Get market depth (order book) for a symbol",
		Long: `Get Level 2 market depth data for a symbol.

Examples:
  longbridge-fs depth AAPL.US
  longbridge-fs depth 700.HK --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("exactly one symbol is required")
			}
			return runDepth(args[0])
		},
	}

	return cmd
}

func runDepth(symbol string) error {
	ctx := context.Background()

	qc, err := createQuoteContext()
	if err != nil {
		return fmt.Errorf("failed to initialize quote context: %w", err)
	}

	// Get security depth
	depth, err := qc.Depth(ctx, symbol)
	if err != nil {
		return fmt.Errorf("failed to fetch depth: %w", err)
	}

	switch outputFormat {
	case "json":
		return outputJSON(depth)
	default:
		return outputDepthTable(depth)
	}
}

func outputDepthTable(depth *quote.SecurityDepth) error {
	fmt.Printf("Market Depth for %s\n\n", depth.Symbol)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Print asks (in reverse order)
	fmt.Fprintln(w, "Asks (Sell Orders):")
	fmt.Fprintln(w, "Price\t\tVolume\t\tOrders")
	fmt.Fprintln(w, "-----\t\t------\t\t------")

	// Reverse iterate asks to show highest price first
	for i := len(depth.Ask) - 1; i >= 0; i-- {
		ask := depth.Ask[i]
		fmt.Fprintf(w, "%.3f\t\t%d\t\t%d\n", decFloat(ask.Price), ask.Volume, ask.OrderNum)
	}

	fmt.Fprintln(w, "\nBids (Buy Orders):")
	fmt.Fprintln(w, "Price\t\tVolume\t\tOrders")
	fmt.Fprintln(w, "-----\t\t------\t\t------")

	for _, bid := range depth.Bid {
		fmt.Fprintf(w, "%.3f\t\t%d\t\t%d\n", decFloat(bid.Price), bid.Volume, bid.OrderNum)
	}

	return w.Flush()
}

func klinesCmd() *cobra.Command {
	var period string
	var count int64

	cmd := &cobra.Command{
		Use:   "klines [symbol]",
		Short: "Get K-line (candlestick) data for a symbol",
		Long: `Get K-line (candlestick) historical data for a symbol.

Supported periods: day, week, month, year, 1m, 5m, 15m, 30m, 60m

Examples:
  longbridge-fs klines AAPL.US --period day --count 30
  longbridge-fs klines 700.HK --period week --count 52 --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("exactly one symbol is required")
			}
			return runKlines(args[0], period, count)
		},
	}

	cmd.Flags().StringVar(&period, "period", "day", "K-line period (day, week, month, year, 1m, 5m, 15m, 30m, 60m)")
	cmd.Flags().Int64Var(&count, "count", 30, "Number of K-lines to fetch")

	return cmd
}

func runKlines(symbol string, periodStr string, count int64) error {
	ctx := context.Background()

	qc, err := createQuoteContext()
	if err != nil {
		return fmt.Errorf("failed to initialize quote context: %w", err)
	}

	// Map period string to SDK period type
	var period quote.Period
	switch strings.ToLower(periodStr) {
	case "day", "d", "1d":
		period = quote.PeriodDay
	case "week", "w", "1w":
		period = quote.PeriodWeek
	case "month", "m", "1M":
		period = quote.PeriodMonth
	case "year", "y", "1y":
		period = quote.PeriodYear
	case "1m", "min1", "minute":
		period = quote.PeriodOneMinute
	case "5m", "min5":
		period = quote.PeriodFiveMinute
	case "15m", "min15":
		period = quote.PeriodFifteenMinute
	case "30m", "min30":
		period = quote.PeriodThirtyMinute
	case "60m", "min60", "hour", "1h":
		period = quote.PeriodSixtyMinute
	default:
		return fmt.Errorf("invalid period: %s", periodStr)
	}

	// Fetch candlesticks (convert int64 to int32)
	candles, err := qc.Candlesticks(ctx, symbol, period, int32(count), quote.AdjustTypeNo)
	if err != nil {
		return fmt.Errorf("failed to fetch candlesticks: %w", err)
	}

	switch outputFormat {
	case "json":
		return outputJSON(candles)
	default:
		return outputKlinesTable(candles, symbol, periodStr)
	}
}

func outputKlinesTable(candles []*quote.Candlestick, symbol, period string) error {
	fmt.Printf("K-Lines for %s (%s)\n\n", symbol, period)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Date\t\tOpen\t\tHigh\t\tLow\t\tClose\t\tVolume\t\tTurnover")
	fmt.Fprintln(w, "----\t\t----\t\t----\t\t---\t\t-----\t\t------\t\t--------")

	for _, c := range candles {
		date := time.Unix(c.Timestamp, 0).Format("2006-01-02 15:04")
		fmt.Fprintf(w, "%s\t\t%.3f\t\t%.3f\t\t%.3f\t\t%.3f\t\t%d\t\t%.0f\n",
			date, decFloat(c.Open), decFloat(c.High), decFloat(c.Low), decFloat(c.Close), c.Volume, decFloat(c.Turnover))
	}

	return w.Flush()
}

func intradayCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "intraday [symbol]",
		Short: "Get intraday trading data for a symbol",
		Long: `Get minute-by-minute intraday trading data for a symbol.

Examples:
  longbridge-fs intraday AAPL.US
  longbridge-fs intraday 700.HK --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("exactly one symbol is required")
			}
			return runIntraday(args[0])
		},
	}

	return cmd
}

func runIntraday(symbol string) error {
	ctx := context.Background()

	qc, err := createQuoteContext()
	if err != nil {
		return fmt.Errorf("failed to initialize quote context: %w", err)
	}

	// Fetch intraday data
	lines, err := qc.Intraday(ctx, symbol)
	if err != nil {
		return fmt.Errorf("failed to fetch intraday data: %w", err)
	}

	switch outputFormat {
	case "json":
		return outputJSON(lines)
	default:
		return outputIntradayTable(lines, symbol)
	}
}

func outputIntradayTable(lines []*quote.IntradayLine, symbol string) error {
	fmt.Printf("Intraday Data for %s\n\n", symbol)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Time\t\tPrice\t\tVolume\t\tTurnover\t\tAvg Price")
	fmt.Fprintln(w, "----\t\t-----\t\t------\t\t--------\t\t---------")

	for _, line := range lines {
		timeStr := time.Unix(line.Timestamp, 0).Format("15:04")
		fmt.Fprintf(w, "%s\t\t%.3f\t\t%d\t\t%.0f\t\t%.3f\n",
			timeStr, decFloat(line.Price), line.Volume, decFloat(line.Turnover), decFloat(line.AvgPrice))
	}

	return w.Flush()
}
