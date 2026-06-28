package alerts

import (
	"context"
	"encoding/json"
	"fmt"
)

type MessageSender interface {
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
	BullSpreadATMThreshold  float64
	BullSpreadOTMThreshold  float64
}

type Engine struct {
	cfg    Config
	sender MessageSender
	store  AlertStore
}

func NewEngine(cfg Config, sender MessageSender, store AlertStore) *Engine {
	return &Engine{cfg: cfg, sender: sender, store: store}
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
	return e.send(ctx, "arbitrage", key, fmt.Sprintf("Arbitrage R=%.2f%% strike=%.0f expiry=%s", input.ReturnPct, input.Strike, input.Expiry))
}

func (e *Engine) MaybeSendArbitrageR12(ctx context.Context, input ArbitrageAlertInput) (bool, error) {
	if e.cfg.ArbitrageR12Threshold <= 0 || input.ReturnPct < e.cfg.ArbitrageR12Threshold {
		return false, nil
	}
	key := fmt.Sprintf("bale-arb-r12:%s:%.0f", input.Expiry, input.Strike)
	msg := fmt.Sprintf("🔔 فرصت آربیتراژ\nنماد: %s\nاسترایک: %.0f\nانقضا: %s\nR'(S×۱.۱۲۵): %.2f%%", input.Symbol, input.Strike, input.Expiry, input.ReturnPct)
	return e.send(ctx, "bale_arb_r12", key, msg)
}

type CoveredCallAlertInput struct {
	Symbol       string
	Expiry       string
	Strike       float64
	StaticROIPct float64
}

func (e *Engine) MaybeSendCoveredCallROI(ctx context.Context, input CoveredCallAlertInput) (bool, error) {
	if e.cfg.CoveredCallROIThreshold <= 0 || input.StaticROIPct < e.cfg.CoveredCallROIThreshold {
		return false, nil
	}
	key := fmt.Sprintf("bale-cc-roi:%s:%.0f", input.Expiry, input.Strike)
	msg := fmt.Sprintf("📈 کاورد کال پربازده\nنماد: %s\nاسترایک: %.0f\nانقضا: %s\nStatic ROI: %.2f%%",
		input.Symbol, input.Strike, input.Expiry, input.StaticROIPct)
	return e.send(ctx, "bale_cc_roi", key, msg)
}

func (e *Engine) MaybeSendBreadth(ctx context.Context, avg float64, state string) (bool, error) {
	if state == "normal" {
		return false, nil
	}
	key := fmt.Sprintf("breadth:%s", state)
	return e.send(ctx, "breadth", key, fmt.Sprintf("Breadth Thrust alert=%s avg10=%.4f", state, avg))
}

func (e *Engine) MaybeSendAdvanceDecline(ctx context.Context, avg float64, state string) (bool, error) {
	if state == "normal" {
		return false, nil
	}
	key := fmt.Sprintf("ad:%s", state)
	return e.send(ctx, "advance_decline", key, fmt.Sprintf("Advance/Decline alert=%s avg10=%.4f", state, avg))
}

type BullSpreadAlertInput struct {
	K1Symbol string
	K2Symbol string
	Expiry   string
	R        float64
	Kind     string // "ATM" or "OTM"
}

func (e *Engine) MaybeSendBullSpreadBale(ctx context.Context, input BullSpreadAlertInput) (bool, error) {
	threshold := e.cfg.BullSpreadATMThreshold
	if input.Kind == "OTM" {
		threshold = e.cfg.BullSpreadOTMThreshold
	}
	if threshold <= 0 || input.R < threshold {
		return false, nil
	}
	key := fmt.Sprintf("bale-bs:%s:%s:%s", input.Kind, input.Expiry, input.K2Symbol)
	msg := fmt.Sprintf("📊 بول کال اسپرد (%s)\n%s / %s\nانقضا: %s\nریوارد/ریسک: %.2f",
		input.Kind, input.K1Symbol, input.K2Symbol, input.Expiry, input.R)
	return e.send(ctx, "bale_bull_spread", key, msg)
}

func (e *Engine) MaybeSendMatrixAlert(ctx context.Context, ruleID string, diff float64, message string) (bool, error) {
	key := fmt.Sprintf("matrix:%s", ruleID)
	return e.send(ctx, "matrix", key, message)
}

func (e *Engine) send(ctx context.Context, alertType, key, message string) (bool, error) {
	if e.sender == nil {
		return false, fmt.Errorf("sender not configured")
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
