package matrixalerts

import (
	"encoding/json"
	"os"
)

type Rule struct {
	ID        string  `json:"id"`
	SymbolA   string  `json:"symbol_a"`
	SymbolB   string  `json:"symbol_b"`
	Operator  string  `json:"operator"`
	Threshold float64 `json:"threshold"`
	Message   string  `json:"message"`
}

func LoadRules(path string) ([]Rule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var rules []Rule
	if err := json.Unmarshal(data, &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

func (r Rule) Evaluate(priceA, priceB float64) (diff float64, triggered bool) {
	diff = priceA - priceB
	switch r.Operator {
	case ">=":
		return diff, diff >= r.Threshold
	case "<=":
		return diff, diff <= r.Threshold
	case "==":
		return diff, diff == r.Threshold
	default:
		return diff, false
	}
}
