package alerts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TelegramSender interface {
	SendMessage(ctx context.Context, text string) error
}

type Config struct {
	ArbitrageRThreshold   float64
	BreadthHighThreshold  float64
	BreadthLowThreshold   float64
	AdvanceHighThreshold  float64
	AdvanceLowThreshold   float64
}

type Engine struct {
	cfg    Config
	sender TelegramSender
	store  *Store
}

func NewEngine(cfg Config, sender TelegramSender, store *Store) *Engine {
	return &Engine{cfg: cfg, sender: sender, store: store}
}

type ArbitrageAlertInput struct {
	Expiry    string
	Strike    float64
	ReturnPct float64
}

func (e *Engine) MaybeSendArbitrage(ctx context.Context, input ArbitrageAlertInput) (bool, error) {
	if e.cfg.ArbitrageRThreshold <= 0 || input.ReturnPct < e.cfg.ArbitrageRThreshold {
		return false, nil
	}
	key := fmt.Sprintf("arb:%s:%.0f:%.2f", input.Expiry, input.Strike, input.ReturnPct)
	return e.send(ctx, "arbitrage", key, fmt.Sprintf("Arbitrage R=%.2f%% strike=%.0f expiry=%s", input.ReturnPct, input.Strike, input.Expiry))
}

func (e *Engine) MaybeSendBreadth(ctx context.Context, avg float64, state string) (bool, error) {
	if state == "normal" {
		return false, nil
	}
	key := fmt.Sprintf("breadth:%s:%.4f", state, avg)
	return e.send(ctx, "breadth", key, fmt.Sprintf("Breadth Thrust alert=%s avg10=%.4f", state, avg))
}

func (e *Engine) MaybeSendAdvanceDecline(ctx context.Context, avg float64, state string) (bool, error) {
	if state == "normal" {
		return false, nil
	}
	key := fmt.Sprintf("ad:%s:%.4f", state, avg)
	return e.send(ctx, "advance_decline", key, fmt.Sprintf("Advance/Decline alert=%s avg10=%.4f", state, avg))
}

func (e *Engine) send(ctx context.Context, alertType, key, message string) (bool, error) {
	if e.sender == nil {
		return false, fmt.Errorf("telegram sender not configured")
	}
	sent, err := e.store.WasSent(ctx, alertType, key)
	if err != nil {
		return false, err
	}
	if sent {
		return false, nil
	}
	if err := e.sender.SendMessage(ctx, message); err != nil {
		return false, err
	}
	payload, _ := json.Marshal(map[string]string{"message": message})
	if err := e.store.Record(ctx, alertType, key, payload); err != nil {
		return false, err
	}
	return true, nil
}

type Store struct {
	pool *pgxpool.Pool
	mem  map[string]struct{}
	mu   sync.Mutex
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, mem: make(map[string]struct{})}
}

func (s *Store) WasSent(ctx context.Context, alertType, key string) (bool, error) {
	digest := hashKey(alertType + ":" + key)
	if s.pool == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		_, ok := s.mem[digest]
		return ok, nil
	}
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM alert_history WHERE alert_type=$1 AND alert_key=$2)`,
		alertType, digest,
	).Scan(&exists)
	return exists, err
}

func (s *Store) Record(ctx context.Context, alertType, key string, payload []byte) error {
	digest := hashKey(alertType + ":" + key)
	if s.pool == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.mem[digest] = struct{}{}
		return nil
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO alert_history (alert_type, alert_key, payload, sent_at) VALUES ($1,$2,$3,$4)
		 ON CONFLICT (alert_type, alert_key) DO NOTHING`,
		alertType, digest, payload, time.Now().UTC(),
	)
	return err
}

func hashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}
