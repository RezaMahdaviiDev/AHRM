package matrixalerts_test

import (
	"os"
	"path/filepath"
	"testing"

	"ahrm/internal/matrixalerts"
)

func TestEvaluateOperators(t *testing.T) {
	rule := matrixalerts.Rule{Operator: ">=", Threshold: 100}
	diff, triggered := rule.Evaluate(500, 300)
	if diff != 200 || !triggered {
		t.Fatalf(">= got diff=%v triggered=%v", diff, triggered)
	}
	diff, triggered = rule.Evaluate(350, 300)
	if triggered {
		t.Fatalf("expected not triggered for diff=%v", diff)
	}

	rule = matrixalerts.Rule{Operator: "<=", Threshold: -50}
	diff, triggered = rule.Evaluate(200, 300)
	if diff != -100 || !triggered {
		t.Fatalf("<= got diff=%v triggered=%v", diff, triggered)
	}

	rule = matrixalerts.Rule{Operator: "==", Threshold: 1200}
	diff, triggered = rule.Evaluate(2700, 1500)
	if diff != 1200 || !triggered {
		t.Fatalf("== got diff=%v triggered=%v", diff, triggered)
	}
}

func TestEvaluateInvalidOperator(t *testing.T) {
	rule := matrixalerts.Rule{Operator: "!=", Threshold: 100}
	_, triggered := rule.Evaluate(500, 300)
	if triggered {
		t.Fatal("expected invalid operator to not trigger")
	}
}

func TestLoadRulesMissingFile(t *testing.T) {
	rules, err := matrixalerts.LoadRules(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	if rules != nil {
		t.Fatalf("expected nil slice, got %v", rules)
	}
}

func TestLoadRulesValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.json")
	content := `[{"id":"r1","symbol_a":"A","symbol_b":"B","operator":">=","threshold":10,"message":"alert"}]`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	rules, err := matrixalerts.LoadRules(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 || rules[0].ID != "r1" {
		t.Fatalf("rules=%v", rules)
	}
}
