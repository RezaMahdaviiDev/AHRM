package matrix_test

import (
	"testing"

	"ahrm/internal/matrix"
	"ahrm/internal/sourcearena"
)

func TestBuildCallMatrix(t *testing.T) {
	engine := matrix.NewEngine()
	opts := []sourcearena.Option{
		{Name: "ضهرم1200", ClosePrice: 1500, ExpiryDate: "1405/12/15"},
		{Name: "ضهرم1300", ClosePrice: 1200, ExpiryDate: "1405/12/15"},
		{Name: "طهرم1200", ClosePrice: 800, ExpiryDate: "1405/12/15"},
	}
	matrices, err := engine.BuildCalls(opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(matrices) != 1 {
		t.Fatalf("matrices=%d", len(matrices))
	}
	m := matrices[0]
	if m.Expiry != "1405/12/15" {
		t.Fatalf("expiry=%q", m.Expiry)
	}
	if len(m.Symbols) != 2 {
		t.Fatalf("symbols=%d", len(m.Symbols))
	}
	if m.Cells[0][1] != 300 {
		t.Fatalf("cell=%v", m.Cells[0][1])
	}
	if m.Cells[0][0] != 0 {
		t.Fatalf("diag=%v", m.Cells[0][0])
	}
}

func TestBuildCallMatrixMultipleExpiries(t *testing.T) {
	engine := matrix.NewEngine()
	opts := []sourcearena.Option{
		{Name: "ضهرم1200", ClosePrice: 1500, ExpiryDate: "1405/12/15"},
		{Name: "ضهرم1300", ClosePrice: 1200, ExpiryDate: "1404/04/01"},
	}
	matrices, err := engine.BuildCalls(opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(matrices) != 2 {
		t.Fatalf("matrices=%d", len(matrices))
	}
	if matrices[0].Expiry != "1404/04/01" {
		t.Fatalf("first expiry=%q", matrices[0].Expiry)
	}
	if matrices[1].Expiry != "1405/12/15" {
		t.Fatalf("second expiry=%q", matrices[1].Expiry)
	}
	if len(matrices[0].Symbols) != 1 || len(matrices[1].Symbols) != 1 {
		t.Fatalf("expected one symbol per expiry matrix")
	}
}

func TestBuildPutMatrix(t *testing.T) {
	engine := matrix.NewEngine()
	opts := []sourcearena.Option{
		{Name: "طهرم1200", ClosePrice: 800, ExpiryDate: "1405/12/15"},
		{Name: "طهرم1300", ClosePrice: 700, ExpiryDate: "1405/12/15"},
	}
	matrices, err := engine.BuildPuts(opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(matrices) != 1 {
		t.Fatalf("matrices=%d", len(matrices))
	}
	m := matrices[0]
	if m.Cells[1][0] != -100 {
		t.Fatalf("cell=%v", m.Cells[1][0])
	}
}
