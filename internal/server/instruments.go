package server

import (
	"encoding/json"
	"os"
)

type Instrument struct {
	InsCode      string `json:"ins_code"`
	Symbol       string `json:"symbol"`
	CompanyName  string `json:"company_name"`
	ISIN         string `json:"isin"`
	InstrumentID string `json:"instrument_id"`
}

func loadInstruments(path string) []Instrument {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []Instrument
	_ = json.Unmarshal(data, &out)
	return out
}
