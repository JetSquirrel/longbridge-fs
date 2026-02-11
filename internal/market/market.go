package market

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/longportapp/openapi-go/quote"
	"github.com/shopspring/decimal"
)

// quoteFile defines a candlestick file to generate.
type quoteFile struct {
	Name   string
	Period quote.Period
	Count  int32
}

// quoteFileMap defines the candlestick files to generate per symbol.
var quoteFileMap = []quoteFile{
	{"D", quote.PeriodDay, 120},
	{"W", quote.PeriodWeek, 52},
	{"M", quote.PeriodMonth, 24},
	{"Y", quote.PeriodYear, 10},
	{"5D", quote.PeriodFiveMinute, 390},
}

// RefreshQuotes scans /quote/track/ for pending refresh requests.
// When a track file exists, we fetch quote data into /quote/hold/{SYMBOL}/,
// then remove the track file (one-shot trigger).
// To re-fetch: touch the track file again.
func RefreshQuotes(ctx context.Context, qc *quote.QuoteContext, root string) {
	trackDir := filepath.Join(root, "quote", "track")
	entries, err := os.ReadDir(trackDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		symbol := e.Name()
		holdSymbolDir := filepath.Join(root, "quote", "hold", symbol)
		if err := os.MkdirAll(holdSymbolDir, 0755); err != nil {
			log.Printf("quote mkdir %s: %v", symbol, err)
			continue
		}
		if err := fetchAndWriteQuote(ctx, qc, holdSymbolDir, symbol); err != nil {
			log.Printf("quote refresh %s: %v", symbol, err)
			continue
		}
		// Remove track file after successful fetch (one-shot)
		os.Remove(filepath.Join(trackDir, symbol))
		log.Printf("quote refreshed: hold/%s/", symbol)
	}
}

// fetchAndWriteQuote fetches all quote data for a symbol and writes to the given dir.
func fetchAndWriteQuote(ctx context.Context, qc *quote.QuoteContext, symbolDir, symbol string) error {
	// 1. Overview (real-time quote)
	if err := writeOverview(ctx, qc, symbolDir, symbol); err != nil {
		log.Printf("  overview %s: %v", symbol, err)
	}

	// 2. Intraday
	if err := writeIntraday(ctx, qc, symbolDir, symbol); err != nil {
		log.Printf("  intraday %s: %v", symbol, err)
	}

	// 3. Candlestick files (D, W, M, Y, 5D)
	for _, qf := range quoteFileMap {
		if err := writeCandlesticks(ctx, qc, symbolDir, symbol, qf.Name, qf.Period, qf.Count); err != nil {
			log.Printf("  %s %s: %v", qf.Name, symbol, err)
		}
	}

	return nil
}

// writeOverview writes overview.txt with real-time quote data.
func writeOverview(ctx context.Context, qc *quote.QuoteContext, dir, symbol string) error {
	quotes, err := qc.Quote(ctx, []string{symbol})
	if err != nil {
		return err
	}
	if len(quotes) == 0 {
		return fmt.Errorf("no quote data for %s", symbol)
	}

	q := quotes[0]
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Symbol: %s\n", symbol))
	sb.WriteString(fmt.Sprintf("Last:   %s\n", decStr(q.LastDone)))
	sb.WriteString(fmt.Sprintf("Open:   %s\n", decStr(q.Open)))
	sb.WriteString(fmt.Sprintf("High:   %s\n", decStr(q.High)))
	sb.WriteString(fmt.Sprintf("Low:    %s\n", decStr(q.Low)))
	sb.WriteString(fmt.Sprintf("Prev:   %s\n", decStr(q.PrevClose)))
	sb.WriteString(fmt.Sprintf("Volume: %d\n", q.Volume))
	sb.WriteString(fmt.Sprintf("Turnover: %s\n", decStr(q.Turnover)))

	if q.PreMarketQuote != nil {
		sb.WriteString(fmt.Sprintf("Pre-Market: %s\n", decStr(q.PreMarketQuote.LastDone)))
	}
	if q.PostMarketQuote != nil {
		sb.WriteString(fmt.Sprintf("Post-Market: %s\n", decStr(q.PostMarketQuote.LastDone)))
	}

	t := time.Unix(q.Timestamp, 0).UTC()
	sb.WriteString(fmt.Sprintf("Updated: %s\n", t.Format(time.RFC3339)))

	return os.WriteFile(filepath.Join(dir, "overview.txt"), []byte(sb.String()), 0644)
}

// writeIntraday writes intraday.txt with today's minute-by-minute data.
func writeIntraday(ctx context.Context, qc *quote.QuoteContext, dir, symbol string) error {
	lines, err := qc.Intraday(ctx, symbol)
	if err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%-20s %-12s %-12s %-12s\n", "Time", "Price", "Volume", "AvgPrice"))
	sb.WriteString(strings.Repeat("-", 60) + "\n")
	for _, l := range lines {
		t := time.Unix(l.Timestamp, 0).UTC().Format("15:04")
		sb.WriteString(fmt.Sprintf("%-20s %-12s %-12d %-12s\n", t, decStr(l.Price), l.Volume, decStr(l.AvgPrice)))
	}

	return os.WriteFile(filepath.Join(dir, "intraday.txt"), []byte(sb.String()), 0644)
}

// writeCandlesticks writes a candlestick file (D.txt, W.txt, etc.).
func writeCandlesticks(ctx context.Context, qc *quote.QuoteContext, dir, symbol, name string, period quote.Period, count int32) error {
	sticks, err := qc.Candlesticks(ctx, symbol, period, count, quote.AdjustTypeNo)
	if err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%-12s %-12s %-12s %-12s %-12s %-12s %-14s\n",
		"Date", "Open", "Close", "High", "Low", "Volume", "Turnover"))
	sb.WriteString(strings.Repeat("-", 90) + "\n")

	for _, s := range sticks {
		t := time.Unix(s.Timestamp, 0).UTC()
		var dateStr string
		if period == quote.PeriodFiveMinute {
			dateStr = t.Format("2006-01-02 15:04")
		} else {
			dateStr = t.Format("2006-01-02")
		}
		sb.WriteString(fmt.Sprintf("%-12s %-12s %-12s %-12s %-12s %-12d %-14s\n",
			dateStr, decStr(s.Open), decStr(s.Close), decStr(s.High), decStr(s.Low), s.Volume, decStr(s.Turnover)))
	}

	return os.WriteFile(filepath.Join(dir, name+".txt"), []byte(sb.String()), 0644)
}

// decStr safely converts a *decimal.Decimal to string, handling nil.
func decStr(d interface{}) string {
	if d == nil {
		return "N/A"
	}
	// Handle nil pointer inside interface
	v := reflect.ValueOf(d)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return "N/A"
	}
	switch val := d.(type) {
	case *decimal.Decimal:
		return val.String()
	case decimal.Decimal:
		return val.String()
	default:
		return fmt.Sprintf("%v", d)
	}
}
