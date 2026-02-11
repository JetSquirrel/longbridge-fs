package ledger

import (
	"os"
	"regexp"
	"strings"

	"longbridge-fs/internal/model"
)

// HeaderRe matches beancount transaction header lines like:
// 2026-02-11 * "ORDER" "BUY NVDA"
var HeaderRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\s+\*\s+"(\w+)"\s+"(.+)"`)

// ParseEntries parses a beancount file into a list of entries.
// Each entry starts with a header line and continues with indented meta lines.
func ParseEntries(path string) ([]model.Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	var entries []model.Entry
	var current *model.Entry

	for _, line := range lines {
		if m := HeaderRe.FindStringSubmatch(line); m != nil {
			if current != nil {
				entries = append(entries, *current)
			}
			current = &model.Entry{
				Type:     m[2],
				Meta:     make(map[string]string),
				RawLines: []string{line},
			}
		} else if current != nil {
			current.RawLines = append(current.RawLines, line)
			if strings.HasPrefix(strings.TrimSpace(line), ";") {
				k, v := ParseMeta(line)
				if k != "" {
					current.Meta[k] = v
				}
			}
		}
	}
	if current != nil {
		entries = append(entries, *current)
	}

	return entries, nil
}

// ParseMeta extracts key-value from a beancount meta comment line like:
//
//	; key: value
func ParseMeta(line string) (string, string) {
	s := strings.TrimSpace(line)
	s = strings.TrimPrefix(s, ";")
	s = strings.TrimSpace(s)
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

// OrderFromEntry extracts a ParsedOrder from an ORDER entry.
func OrderFromEntry(e model.Entry) model.ParsedOrder {
	o := model.ParsedOrder{
		IntentID:  e.Meta["intent_id"],
		Side:      strings.ToUpper(e.Meta["side"]),
		Symbol:    e.Meta["symbol"],
		Qty:       e.Meta["qty"],
		OrderType: strings.ToUpper(e.Meta["type"]),
		TIF:       strings.ToUpper(e.Meta["tif"]),
		Price:     e.Meta["price"],
		Market:    e.Meta["market"],
	}
	if o.Market == "" {
		o.Market = "US"
	}
	if o.TIF == "" {
		o.TIF = "DAY"
	}
	if o.OrderType == "" {
		o.OrderType = "MARKET"
	}
	return o
}

// FullSymbol returns a symbol with market suffix, e.g. "NVDA" -> "NVDA.US"
func FullSymbol(symbol, market string) string {
	if strings.Contains(symbol, ".") {
		return symbol
	}
	return symbol + "." + market
}

// BuildLedgerState scans entries and returns a set of intent_ids that already
// have an EXECUTION or REJECTION, plus all ORDER entries.
func BuildLedgerState(entries []model.Entry) (processed map[string]bool, orders []model.Entry) {
	processed = make(map[string]bool)
	for _, e := range entries {
		switch e.Type {
		case "EXECUTION", "REJECTION":
			if id := e.Meta["intent_id"]; id != "" {
				processed[id] = true
			}
		case "ORDER":
			orders = append(orders, e)
		}
	}
	return
}
