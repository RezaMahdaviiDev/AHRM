package matrix_test

import (
	"testing"

	"ahrm/internal/matrix"
	"ahrm/internal/sourcearena"
)

func TestBuildCallMatrix(t *testing.T) {
	engine := matrix.NewEngine()
	opts := []sourcearena.Option{
		{Name: "ضهرم1200", ClosePrice: 1500},
		{Name: "ضهرم1300", ClosePrice: 1200},
		{Name: "طهرم1200", ClosePrice: 800},
	}
	m, err := engine.BuildCalls(opts)
	if err != nil {
		t.Fatal(err)
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

func TestBuildPutMatrix(t *testing.T) {
	engine := matrix.NewEngine()
	opts := []sourcearena.Option{
		{Name: "طهرم1200", ClosePrice: 800},
		{Name: "طهرم1300", ClosePrice: 700},
	}
	m, err := engine.BuildPuts(opts)
	if err != nil {
		t.Fatal(err)
	}
	if m.Cells[1][0] != -100 {
		t.Fatalf("cell=%v", m.Cells[1][0])
	}
}
