package ledger

import (
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"longbridge-fs/internal/model"
)

// CompactBlocks finds completed ORDER+EXECUTION pairs in the beancount ledger,
// moves them into a block under /trade/blocks/{BLOCK_ID}/, and rewrites the ledger.
func CompactBlocks(root string, count int) error {
	bcPath := filepath.Join(root, "trade", "beancount.txt")
	entries, err := ParseEntries(bcPath)
	if err != nil {
		return err
	}

	// Find completed pairs: an ORDER that has a matching EXECUTION or REJECTION
	processed := make(map[string]bool)
	for _, e := range entries {
		if e.Type == "EXECUTION" || e.Type == "REJECTION" {
			if id := e.Meta["intent_id"]; id != "" {
				processed[id] = true
			}
		}
	}

	// Collect entries to compact (ORDER + its EXECUTION/REJECTION)
	var toCompact []model.Entry
	compactedIDs := make(map[string]bool)
	for _, e := range entries {
		id := e.Meta["intent_id"]
		if id == "" {
			continue
		}
		switch e.Type {
		case "ORDER":
			if processed[id] {
				toCompact = append(toCompact, e)
				compactedIDs[id] = true
			}
		case "EXECUTION", "REJECTION":
			if compactedIDs[id] {
				toCompact = append(toCompact, e)
			}
		}
	}

	if len(toCompact) == 0 {
		return nil
	}

	// Build block data
	var dataBuf strings.Builder
	var intentIDs []string
	seen := make(map[string]bool)
	for _, e := range toCompact {
		dataBuf.WriteString(strings.Join(e.RawLines, "\n"))
		dataBuf.WriteString("\n")
		if id := e.Meta["intent_id"]; id != "" && !seen[id] {
			intentIDs = append(intentIDs, id)
			seen[id] = true
		}
	}
	data := dataBuf.String()

	// Block ID: timestamp + sha256 prefix
	now := time.Now()
	hash := sha256.Sum256([]byte(data))
	hashHex := fmt.Sprintf("%x", hash)
	blockID := fmt.Sprintf("%s-%s", now.Format("20060102T150405"), hashHex[:8])

	// Write block
	blockDir := filepath.Join(root, "trade", "blocks", blockID)
	if err := os.MkdirAll(blockDir, 0755); err != nil {
		return err
	}

	// meta.txt
	meta := fmt.Sprintf("block_id: %s\ncreated_at: %s\nentries: %d\nintent_ids: %s\nsha256: %s\n",
		blockID,
		now.UTC().Format(time.RFC3339),
		len(toCompact),
		strings.Join(intentIDs, ", "),
		hashHex,
	)
	if err := os.WriteFile(filepath.Join(blockDir, "meta.txt"), []byte(meta), 0644); err != nil {
		return err
	}

	// data
	if err := os.WriteFile(filepath.Join(blockDir, "data"), []byte(data), 0644); err != nil {
		return err
	}

	// Rewrite ledger without compacted entries
	var remaining []string
	remaining = append(remaining, "; beancount append-only trade ledger")
	remaining = append(remaining, fmt.Sprintf("; compacted to block %s at %s", blockID, now.UTC().Format(time.RFC3339)))
	remaining = append(remaining, "")

	for _, e := range entries {
		id := e.Meta["intent_id"]
		if id != "" && compactedIDs[id] {
			continue // skip compacted entries
		}
		remaining = append(remaining, strings.Join(e.RawLines, "\n"))
	}

	newContent := strings.Join(remaining, "\n") + "\n"
	if err := os.WriteFile(bcPath, []byte(newContent), 0644); err != nil {
		return err
	}

	log.Printf("compacted %d entries into block %s", len(toCompact), blockID)
	return nil
}
