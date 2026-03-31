package research

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"longbridge-fs/internal/model"
)

// GenerateSummary scans research feeds and generates summary.json
func GenerateSummary(root string) error {
	summary := model.ResearchSummary{
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Symbols:   make(map[string]model.SymbolResearch),
	}

	// Scan news feeds
	newsPath := filepath.Join(root, "research", "feeds", "news")
	if entries, err := os.ReadDir(newsPath); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			symbol := entry.Name()
			latest := filepath.Join(newsPath, symbol, "latest.json")
			if _, err := os.Stat(latest); err == nil {
				sr := summary.Symbols[symbol]
				sr.HasNews = true
				summary.Symbols[symbol] = sr
			}
		}
	}

	// Scan topics feeds
	topicsPath := filepath.Join(root, "research", "feeds", "topics")
	if entries, err := os.ReadDir(topicsPath); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			symbol := entry.Name()
			latest := filepath.Join(topicsPath, symbol, "latest.json")
			if _, err := os.Stat(latest); err == nil {
				sr := summary.Symbols[symbol]
				sr.HasTopics = true
				summary.Symbols[symbol] = sr
			}
		}
	}

	// Check quote availability
	quotePath := filepath.Join(root, "quote", "hold")
	if entries, err := os.ReadDir(quotePath); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			symbol := entry.Name()
			overview := filepath.Join(quotePath, symbol, "overview.json")
			if _, err := os.Stat(overview); err == nil {
				sr := summary.Symbols[symbol]
				sr.HasQuote = true
				summary.Symbols[symbol] = sr
			}
		}
	}

	// Scan custom feeds
	customPath := filepath.Join(root, "research", "feeds", "custom")
	if entries, err := os.ReadDir(customPath); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			// Custom feeds are tracked globally, not per-symbol
			// For simplicity, we'll just note their existence
			// Could enhance this to parse and extract symbol references
		}
	}

	// Write summary
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(root, "research", "summary.json"), append(data, '\n'), 0644)
}
