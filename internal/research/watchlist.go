package research

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"longbridge-fs/internal/model"
)

// ParseWatchlist reads and parses research/watchlist.json
func ParseWatchlist(root string) (*model.Watchlist, error) {
	watchlistPath := filepath.Join(root, "research", "watchlist.json")
	data, err := os.ReadFile(watchlistPath)
	if err != nil {
		return nil, err
	}

	var watchlist model.Watchlist
	if err := json.Unmarshal(data, &watchlist); err != nil {
		return nil, fmt.Errorf("parse watchlist: %w", err)
	}

	return &watchlist, nil
}

// WriteWatchlist writes a watchlist to research/watchlist.json
func WriteWatchlist(root string, watchlist *model.Watchlist) error {
	researchDir := filepath.Join(root, "research")
	if err := os.MkdirAll(researchDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(watchlist, "", "  ")
	if err != nil {
		return err
	}

	watchlistPath := filepath.Join(researchDir, "watchlist.json")
	return os.WriteFile(watchlistPath, append(data, '\n'), 0644)
}
