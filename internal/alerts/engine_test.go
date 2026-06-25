package alerts_test

import (
	"context"
	"testing"

	"ahrm/internal/alerts"
)

type fakeSender struct {
	messages []string
}

func (f *fakeSender) SendMessage(_ context.Context, text string) error {
	f.messages = append(f.messages, text)
	return nil
}

func TestDuplicateProtectionWithoutDB(t *testing.T) {
	sender := &fakeSender{}
	engine := alerts.NewEngine(alerts.Config{ArbitrageRThreshold: 1}, sender, alerts.NewMemStore())
	input := alerts.ArbitrageAlertInput{Expiry: "1404/09/15", Strike: 12000, ReturnPct: 5}
	sent, err := engine.MaybeSendArbitrage(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if !sent {
		t.Fatal("expected alert sent")
	}
	sent, err = engine.MaybeSendArbitrage(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if sent {
		t.Fatal("expected duplicate suppressed")
	}
	if len(sender.messages) != 1 {
		t.Fatalf("messages=%d", len(sender.messages))
	}
}

func TestThresholdCheck(t *testing.T) {
	sender := &fakeSender{}
	engine := alerts.NewEngine(alerts.Config{ArbitrageRThreshold: 10}, sender, alerts.NewMemStore())
	sent, err := engine.MaybeSendArbitrage(context.Background(), alerts.ArbitrageAlertInput{ReturnPct: 1})
	if err != nil {
		t.Fatal(err)
	}
	if sent {
		t.Fatal("expected no alert below threshold")
	}
}

func TestMatrixAlertDuplicateProtection(t *testing.T) {
	sender := &fakeSender{}
	engine := alerts.NewEngine(alerts.Config{}, sender, alerts.NewMemStore())
	sent, err := engine.MaybeSendMatrixAlert(context.Background(), "rule1", 1200, "test alert")
	if err != nil {
		t.Fatal(err)
	}
	if !sent {
		t.Fatal("expected alert sent")
	}
	sent, err = engine.MaybeSendMatrixAlert(context.Background(), "rule1", 1200, "test alert")
	if err != nil {
		t.Fatal(err)
	}
	if sent {
		t.Fatal("expected duplicate suppressed")
	}
	if len(sender.messages) != 1 {
		t.Fatalf("messages=%d", len(sender.messages))
	}
}
