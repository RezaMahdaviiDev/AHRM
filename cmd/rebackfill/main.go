package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"ahrm/internal/config"
	"ahrm/internal/market"
	"ahrm/internal/sourcearena"
)

func projectRoot() string {
	if root := os.Getenv("AHRM_ROOT"); root != "" {
		return root
	}
	return "/root/AHRM"
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	store, err := market.NewSQLiteStore(filepath.Join(projectRoot(), "data", "market.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	if !cfg.SourceArena.Configured() {
		log.Fatal("sourcearena not configured")
	}
	client := sourcearena.NewClient(cfg.SourceArena, sourcearena.NopRawStore{})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

	symbols, err := client.FetchAllSymbols(ctx)
	if err != nil {
		log.Fatal(err)
	}

	loc, _ := time.LoadLocation("Asia/Tehran")
	today := time.Now().In(loc).Format("2006-01-02")
	preserve := map[string]struct{}{today: {}}

	log.Printf("rebackfill: symbols=%d preserve=%s", len(symbols), today)
	if err := market.BackfillHistory(ctx, client, symbols, store, preserve, nil); err != nil {
		log.Fatal(err)
	}
	log.Print("rebackfill: done")
}
