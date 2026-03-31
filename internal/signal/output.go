package signal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"longbridge-fs/internal/model"
)

// WriteOutput writes signal output to signal/output/{SYMBOL}/latest.json
func WriteOutput(root, symbol string, output *model.SignalOutput) error {
	outDir := filepath.Join(root, "signal", "output", symbol)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal signal output: %w", err)
	}

	return os.WriteFile(filepath.Join(outDir, "latest.json"), append(data, '\n'), 0644)
}

// AppendHistory appends a signal output snapshot to signal/output/{SYMBOL}/history.jsonl
func AppendHistory(root, symbol string, output *model.SignalOutput) error {
	outDir := filepath.Join(root, "signal", "output", symbol)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	line, err := json.Marshal(output)
	if err != nil {
		return fmt.Errorf("marshal signal output: %w", err)
	}

	f, err := os.OpenFile(filepath.Join(outDir, "history.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "%s\n", line)
	return err
}

// WriteActiveSignals writes the aggregated active signals to signal/active.json
func WriteActiveSignals(root string, active *model.ActiveSignals) error {
	data, err := json.MarshalIndent(active, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal active signals: %w", err)
	}

	return os.WriteFile(filepath.Join(root, "signal", "active.json"), append(data, '\n'), 0644)
}
