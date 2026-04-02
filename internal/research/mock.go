package research

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"longbridge-fs/internal/model"
)

// mockNewsTitles contains sample news headlines for mock mode
var mockNewsTitles = []string{
	"Strong earnings beat analyst expectations",
	"New product launch drives analyst upgrades",
	"Market volatility creates buying opportunity",
	"Institutional investors increase stake",
	"Revenue growth accelerates in latest quarter",
	"Strategic partnership announced with major player",
	"Cost-cutting measures improve margins",
	"Expansion into new markets on track",
}

// mockTopicTitles contains sample discussion topics for mock mode
var mockTopicTitles = []string{
	"Is now a good time to buy?",
	"Earnings preview: what to expect",
	"Technical analysis shows bullish signal",
	"Comparing sector peers performance",
	"Long-term growth thesis intact",
	"Short-term headwinds vs long-term opportunity",
}

// RefreshFeedsMock generates synthetic research feeds without real API calls.
// Used in mock mode to enable full pipeline simulation.
func RefreshFeedsMock(root string) error {
	watchlist, err := ParseWatchlist(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No watchlist, skip
		}
		return fmt.Errorf("parse watchlist: %w", err)
	}

	if len(watchlist.Symbols) == 0 {
		return nil
	}

	for _, symbol := range watchlist.Symbols {
		for _, feed := range watchlist.Feeds {
			switch feed {
			case "news":
				if err := writeMockNews(root, symbol); err != nil {
					fmt.Printf("Warning: failed to write mock news for %s: %v\n", symbol, err)
				}
			case "topics":
				if err := writeMockTopics(root, symbol); err != nil {
					fmt.Printf("Warning: failed to write mock topics for %s: %v\n", symbol, err)
				}
			}
		}
	}

	return GenerateSummary(root)
}

// writeMockNews writes synthetic news feed data for a symbol
func writeMockNews(root, symbol string) error {
	now := time.Now().UTC()
	// Pick a couple of headlines deterministically based on symbol
	idx := len(symbol) % len(mockNewsTitles)
	items := []model.NewsItem{
		{
			ID:          fmt.Sprintf("mock-news-%s-001", symbol),
			Title:       fmt.Sprintf("[%s] %s", symbol, mockNewsTitles[idx]),
			Source:      "mock",
			PublishedAt: now.Add(-2 * time.Hour).Format(time.RFC3339),
			Summary:     fmt.Sprintf("Mock news item for %s generated at %s", symbol, now.Format(time.RFC3339)),
		},
		{
			ID:          fmt.Sprintf("mock-news-%s-002", symbol),
			Title:       fmt.Sprintf("[%s] %s", symbol, mockNewsTitles[(idx+1)%len(mockNewsTitles)]),
			Source:      "mock",
			PublishedAt: now.Add(-5 * time.Hour).Format(time.RFC3339),
			Summary:     fmt.Sprintf("Additional mock news item for %s", symbol),
		},
	}

	feed := model.NewsFeed{
		Symbol:    symbol,
		FetchedAt: now.Format(time.RFC3339),
		Items:     items,
	}

	newsDir := filepath.Join(root, "research", "feeds", "news", symbol)
	if err := os.MkdirAll(newsDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(feed, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(newsDir, "latest.json"), append(data, '\n'), 0644)
}

// writeMockTopics writes synthetic topics feed data for a symbol
func writeMockTopics(root, symbol string) error {
	now := time.Now().UTC()
	idx := (len(symbol) + 1) % len(mockTopicTitles)
	items := []model.TopicItem{
		{
			ID:          fmt.Sprintf("mock-topic-%s-001", symbol),
			Title:       fmt.Sprintf("[%s] %s", symbol, mockTopicTitles[idx]),
			Source:      "mock",
			PublishedAt: now.Add(-1 * time.Hour).Format(time.RFC3339),
			Summary:     fmt.Sprintf("Mock community discussion for %s", symbol),
		},
	}

	feed := model.TopicsFeed{
		Symbol:    symbol,
		FetchedAt: now.Format(time.RFC3339),
		Items:     items,
	}

	topicsDir := filepath.Join(root, "research", "feeds", "topics", symbol)
	if err := os.MkdirAll(topicsDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(feed, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(topicsDir, "latest.json"), append(data, '\n'), 0644)
}

// GenerateMockKlineData writes synthetic daily kline data for a symbol.
// Used in mock mode to enable signal computation without real quote data.
func GenerateMockKlineData(root, symbol string, days int) error {
	holdDir := filepath.Join(root, "quote", "hold", symbol)
	if err := os.MkdirAll(holdDir, 0755); err != nil {
		return err
	}

	klinePath := filepath.Join(holdDir, "D.json")
	// Only generate if file doesn't exist yet
	if _, err := os.Stat(klinePath); err == nil {
		return nil
	}

	bars := make([]model.Candlestick, days)
	basePrice := 100.0 + float64(rand.Intn(400)) // random base 100–500
	now := time.Now().UTC()

	for i := 0; i < days; i++ {
		day := now.AddDate(0, 0, -(days - 1 - i))
		// simple random walk with a slight upward drift (bias of 0.02% per day)
		change := (rand.Float64() - 0.48) * 2.0
		basePrice = basePrice * (1 + change/100)
		if basePrice < 1 {
			basePrice = 1
		}
		open := basePrice * (1 + (rand.Float64()-0.5)*0.01)
		high := basePrice * (1 + rand.Float64()*0.02)
		low := basePrice * (1 - rand.Float64()*0.02)
		close_ := basePrice * (1 + (rand.Float64()-0.5)*0.01)
		volume := int64(100000 + rand.Intn(900000))
		bars[i] = model.Candlestick{
			Date:     day.Format("2006-01-02"),
			Open:     round2(open),
			Close:    round2(close_),
			High:     round2(high),
			Low:      round2(low),
			Volume:   volume,
			Turnover: round2(float64(volume) * close_),
		}
	}

	data, err := json.MarshalIndent(bars, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(klinePath, append(data, '\n'), 0644)
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
