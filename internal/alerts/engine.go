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
	ArbitrageRThreshold     float64
	ArbitrageR12Threshold   float64
	BreadthHighThreshold    float64
	BreadthLowThreshold     float64
	AdvanceHighThreshold    float64
	AdvanceLowThreshold     float64
	CoveredCallROIThreshold float64
}

type Engine struct {
	cfg        Config
	sender     TelegramSender
	baleSender TelegramSender
	store      *Store
}

func NewEngine(cfg Config, sender TelegramSender, baleSender TelegramSender, store *Store) *Engine {
	return &Engine{cfg: cfg, sender: sender, baleSender: baleSender, store: store}
}

type ArbitrageAlertInput struct {
	Symbol    string
	Expiry    string
	Strike    float64
	ReturnPct float64
}

func (e *Engine) MaybeSendArbitrage(ctx context.Context, input ArbitrageAlertInput) (bool, error) {
	if e.cfg.ArbitrageRThreshold <= 0 || input.ReturnPct < e.cfg.ArbitrageRThreshold {
		return false, nil
	}
	key := fmt.Sprintf("arb:%s:%.0f:%.2f", input.Expiry, input.Strike, input.ReturnPct)
	return e.sendVia(ctx, e.sender, "arbitrage", key, fmt.Sprintf("Arbitrage R=%.2f%% strike=%.0f expiry=%s", input.ReturnPct, input.Strike, input.Expiry))
}

func (e *Engine) MaybeSendArbitrageR12Bale(ctx context.Context, input ArbitrageAlertInput) (bool, error) {
	if e.cfg.ArbitrageR12Threshold <= 0 || input.ReturnPct < e.cfg.ArbitrageR12Threshold {
		return false, nil
	}
	key := fmt.Sprintf("bale-arb-r12:%s:%.0f:%.2f", input.Expiry, input.Strike, input.ReturnPct)
	msg := fmt.Sprintf("🔔 فرصت آربیتراژ\nنماد: %s\nاسترایک: %.0f\nانقضا: %s\nR'(S×۱.۱۲۵): %.2f%%", input.Symbol, input.Strike, input.Expiry, input.ReturnPct)
	return e.sendVia(ctx, e.baleSender, "bale_arb_r12", key, msg)
}

type CoveredCallAlertInput struct {
	Symbol       string
	Expiry       string
	Strike       float64
	StaticROIPct float64
}

func (e *Engine) MaybeSendCoveredCallROIBale(ctx context.Context, input CoveredCallAlertInput) (bool, error) {
	if e.cfg.CoveredCallROIThreshold <= 0 || input.StaticROIPct < e.cfg.CoveredCallROIThreshold {
		return false, nil
	}
	key := fmt.Sprintf("bale-cc-roi:%s:%.0f:%.2f", input.Expiry, input.Strike, input.StaticROIPct)
	msg := fmt.Sprintf("📈 کاورد کال پربازده\nنماد: %s\nاسترایک: %.0f\nانقضا: %s\nStatic ROI: %.2f%%",
		input.Symbol, input.Strike, input.Expiry, input.StaticROIPct)
	return e.sendVia(ctx, e.baleSender, "bale_cc_roi", key, msg)
}

func (e *Engine) MaybeSendBreadth(ctx context.Context, avg float64, state string) (bool, error) {
	if state == "normal" {
		return false, nil
	}
	key := fmt.Sprintf("breadth:%s:%.4f", state, avg)
	return e.sendVia(ctx, e.sender, "breadth", key, fmt.Sprintf("Breadth Thrust alert=%s avg10=%.4f", state, avg))
}

func (e *Engine) MaybeSendAdvanceDecline(ctx context.Context, avg float64, state string) (bool, error) {
	if state == "normal" {
		return false, nil
	}
	key := fmt.Sprintf("ad:%s:%.4f", state, avg)
	return e.sendVia(ctx, e.sender, "advance_decline", key, fmt.Sprintf("Advance/Decline alert=%s avg10=%.4f", state, avg))
}

func (e *Engine) MaybeSendMatrixAlert(ctx context.Context, ruleID string, diff float64, message string) (bool, error) {
	key := fmt.Sprintf("matrix:%s:%.0f", ruleID, diff)
	return e.sendVia(ctx, e.sender, "matrix", key, message)
}

func (e *Engine) send(ctx context.Context, alertType, key, message string) (bool, error) {
	return e.sendVia(ctx, e.sender, alertType, key, message)
}

func (e *Engine) sendVia(ctx context.Context, sender TelegramSender, alertType, key, message string) (bool, error) {
	if sender == nil {
		return false, fmt.Errorf("sender not configured")
	}
	sent, err := e.store.WasSent(ctx, alertType, key)
	if err != nil {
		return false, err
	}
	if sent {
		return false, nil
	}
	if err := sender.SendMessage(ctx, message); err != nil {
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
