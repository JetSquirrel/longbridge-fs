package signal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"longbridge-fs/internal/model"
)

// ListDefinitions scans signal/definitions/ and returns all enabled signal definitions.
func ListDefinitions(root string) ([]*model.SignalDefinition, error) {
	defsDir := filepath.Join(root, "signal", "definitions")

	entries, err := os.ReadDir(defsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read definitions dir: %w", err)
	}

	var defs []*model.SignalDefinition
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		def, err := ParseDefinition(filepath.Join(defsDir, entry.Name()))
		if err != nil {
			// Skip invalid definitions but log
			fmt.Printf("Warning: skipping invalid signal definition %s: %v\n", entry.Name(), err)
			continue
		}

		if def.Enabled {
			defs = append(defs, def)
		}
	}

	return defs, nil
}

// ParseDefinition reads and parses a single signal definition file.
func ParseDefinition(path string) (*model.SignalDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var def model.SignalDefinition
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("parse definition: %w", err)
	}

	if def.Name == "" {
		return nil, fmt.Errorf("definition missing name")
	}

	if def.Type != "builtin" && def.Type != "external" {
		return nil, fmt.Errorf("unknown definition type %q (must be builtin or external)", def.Type)
	}

	return &def, nil
}

// paramFloat extracts a float64 parameter from a signal definition's Params map.
// Returns defaultVal if the key is absent or not numeric.
func paramFloat(params map[string]interface{}, key string, defaultVal float64) float64 {
	if params == nil {
		return defaultVal
	}
	v, ok := params[key]
	if !ok {
		return defaultVal
	}
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	}
	return defaultVal
}

// paramInt extracts an int parameter from a signal definition's Params map.
func paramInt(params map[string]interface{}, key string, defaultVal int) int {
	f := paramFloat(params, key, float64(defaultVal))
	return int(f)
}
