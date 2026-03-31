package research

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"longbridge-fs/internal/credential"
	"longbridge-fs/internal/model"

	"github.com/longbridge/openapi-go/content"
)

// RefreshFeeds reads the watchlist and refreshes news/topics feeds from Content API
func RefreshFeeds(ctx context.Context, root, credFile string) error {
	// Parse watchlist
	watchlist, err := ParseWatchlist(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No watchlist, skip refresh
		}
		return fmt.Errorf("parse watchlist: %w", err)
	}

	// Load credentials
	cfg, err := credential.Load(credFile)
	if err != nil {
		return fmt.Errorf("load credentials: %w", err)
	}

	// Create content context
	cc, err := content.NewFromCfg(cfg)
	if err != nil {
		return fmt.Errorf("create content context: %w", err)
	}

	// Refresh feeds for each symbol
	for _, symbol := range watchlist.Symbols {
		for _, feed := range watchlist.Feeds {
			switch feed {
			case "news":
				if err := refreshNews(ctx, cc, root, symbol); err != nil {
					// Log error but continue with other symbols
					fmt.Printf("Warning: failed to refresh news for %s: %v\n", symbol, err)
				}
			case "topics":
				if err := refreshTopics(ctx, cc, root, symbol); err != nil {
					fmt.Printf("Warning: failed to refresh topics for %s: %v\n", symbol, err)
				}
			}
		}
	}

	// Generate summary
	return GenerateSummary(root)
}

// refreshNews fetches and writes news feed for a symbol
func refreshNews(ctx context.Context, cc *content.ContentContext, root, symbol string) error {
	newsItems, err := cc.News(ctx, symbol)
	if err != nil {
		return err
	}

	feed := model.NewsFeed{
		Symbol:    symbol,
		FetchedAt: time.Now().UTC().Format(time.RFC3339),
		Items:     make([]model.NewsItem, 0, len(newsItems)),
	}

	for _, item := range newsItems {
		feed.Items = append(feed.Items, model.NewsItem{
			ID:          item.Id,
			Title:       item.Title,
			Source:      "longbridge", // Content API doesn't provide source in the type
			PublishedAt: item.PublishedAt.Format(time.RFC3339),
			Summary:     item.Description,
			URL:         item.Url,
		})
	}

	// Write to feeds/news/{SYMBOL}/latest.json
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

// refreshTopics fetches and writes topics feed for a symbol
func refreshTopics(ctx context.Context, cc *content.ContentContext, root, symbol string) error {
	topicItems, err := cc.Topics(ctx, symbol)
	if err != nil {
		return err
	}

	feed := model.TopicsFeed{
		Symbol:    symbol,
		FetchedAt: time.Now().UTC().Format(time.RFC3339),
		Items:     make([]model.TopicItem, 0, len(topicItems)),
	}

	for _, item := range topicItems {
		feed.Items = append(feed.Items, model.TopicItem{
			ID:          item.Id,
			Title:       item.Title,
			Source:      "longbridge", // Content API doesn't provide source in the type
			PublishedAt: item.PublishedAt.Format(time.RFC3339),
			Summary:     item.Description,
			URL:         item.Url,
		})
	}

	// Write to feeds/topics/{SYMBOL}/latest.json
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
