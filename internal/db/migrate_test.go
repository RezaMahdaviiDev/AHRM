package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListMigrationFilesEmptyDir(t *testing.T) {
	dir := t.TempDir()
	files, err := listMigrationFiles(dir)
	if err != nil {
		t.Fatalf("listMigrationFiles() error = %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected 0 files, got %d", len(files))
	}
}

func TestListMigrationFilesSorted(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"002_second.up.sql", "001_first.up.sql"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("-- noop"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	files, err := listMigrationFiles(dir)
	if err != nil {
		t.Fatalf("listMigrationFiles() error = %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	if files[0].version != 1 || files[1].version != 2 {
		t.Fatalf("unexpected order: %+v", files)
	}
}

func TestListMigrationFilesInvalidName(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bad.up.sql"), []byte("-- noop"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := listMigrationFiles(dir); err == nil {
		t.Fatal("expected error for invalid migration filename")
	}
}
