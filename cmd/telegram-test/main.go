package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"ahrm/internal/config"
	"ahrm/internal/telegram"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}
	if !cfg.Telegram.Configured() {
		fmt.Println("Telegram not fully configured (need TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID)")
		os.Exit(1)
	}
	client := telegram.NewClient(cfg.Telegram)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	msg := "AHRM Scanner — تست موفق ✅"
	if err := client.SendMessage(ctx, msg); err != nil {
		fmt.Fprintf(os.Stderr, "send failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Telegram test message sent OK")
}
