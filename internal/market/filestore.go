package market

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"ahrm/internal/indicators"
)

const maxHistoryDays = 30

type fileRecord struct {
	Date     string `json:"date"`
	Positive int    `json:"positive"`
	Negative int    `json:"negative"`
	Total    int    `json:"total"`
}

type FileStore struct {
	path string
	mu   sync.Mutex
}

func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

func (s *FileStore) UpsertToday(_ context.Context, day indicators.DailyMarket) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	records := s.load()
	today := time.Now().UTC().Format("2006-01-02")

	found := false
	for i, r := range records {
		if r.Date == today {
			records[i].Positive = day.Positive
			records[i].Negative = day.Negative
			records[i].Total = day.Total
			found = true
			break
		}
	}
	if !found {
		records = append(records, fileRecord{
			Date:     today,
			Positive: day.Positive,
			Negative: day.Negative,
			Total:    day.Total,
		})
	}

	if len(records) > maxHistoryDays {
		records = records[len(records)-maxHistoryDays:]
	}

	return s.save(records)
}

func (s *FileStore) UpsertDay(_ context.Context, date time.Time, day indicators.DailyMarket) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	records := s.load()
	dateStr := date.UTC().Format("2006-01-02")

	found := false
	for i, r := range records {
		if r.Date == dateStr {
			records[i].Positive = day.Positive
			records[i].Negative = day.Negative
			records[i].Total = day.Total
			found = true
			break
		}
	}
	if !found {
		records = append(records, fileRecord{
			Date:     dateStr,
			Positive: day.Positive,
			Negative: day.Negative,
			Total:    day.Total,
		})
	}

	// sort by date ascending
	for i := 1; i < len(records); i++ {
		for j := i; j > 0 && records[j].Date < records[j-1].Date; j-- {
			records[j], records[j-1] = records[j-1], records[j]
		}
	}
	if len(records) > maxHistoryDays {
		records = records[len(records)-maxHistoryDays:]
	}
	return s.save(records)
}

func (s *FileStore) LastDays(_ context.Context, days int) ([]indicators.DailyMarket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records := s.load()
	if len(records) > days {
		records = records[len(records)-days:]
	}

	out := make([]indicators.DailyMarket, len(records))
	for i, r := range records {
		out[i] = indicators.DailyMarket{
			Positive: r.Positive,
			Negative: r.Negative,
			Total:    r.Total,
		}
	}
	return out, nil
}

func (s *FileStore) load() []fileRecord {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil
	}
	var records []fileRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil
	}
	return records
}

func (s *FileStore) save(records []fileRecord) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}
