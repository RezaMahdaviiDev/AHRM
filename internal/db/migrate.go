package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

const bootstrapSQL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version BIGINT PRIMARY KEY,
	name TEXT NOT NULL,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`

type migrationFile struct {
	version int64
	name    string
	path    string
}

func Migrate(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	if _, err := pool.Exec(ctx, bootstrapSQL); err != nil {
		return fmt.Errorf("bootstrap schema_migrations: %w", err)
	}

	files, err := listMigrationFiles(dir)
	if err != nil {
		return err
	}

	applied, err := loadAppliedVersions(ctx, pool)
	if err != nil {
		return err
	}

	for _, file := range files {
		if applied[file.version] {
			continue
		}
		sql, readErr := os.ReadFile(file.path)
		if readErr != nil {
			return fmt.Errorf("read migration %s: %w", file.path, readErr)
		}
		tx, beginErr := pool.Begin(ctx)
		if beginErr != nil {
			return fmt.Errorf("begin migration %d: %w", file.version, beginErr)
		}
		if _, execErr := tx.Exec(ctx, string(sql)); execErr != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply migration %d: %w", file.version, execErr)
		}
		if _, insertErr := tx.Exec(ctx,
			`INSERT INTO schema_migrations (version, name) VALUES ($1, $2)`,
			file.version, file.name,
		); insertErr != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record migration %d: %w", file.version, insertErr)
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			return fmt.Errorf("commit migration %d: %w", file.version, commitErr)
		}
	}

	return nil
}

func listMigrationFiles(dir string) ([]migrationFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}

	var files []migrationFile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		base := strings.TrimSuffix(entry.Name(), ".up.sql")
		parts := strings.SplitN(base, "_", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid migration filename: %s", entry.Name())
		}
		version, parseErr := strconv.ParseInt(parts[0], 10, 64)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid migration version in %s: %w", entry.Name(), parseErr)
		}
		files = append(files, migrationFile{
			version: version,
			name:    parts[1],
			path:    filepath.Join(dir, entry.Name()),
		})
	}

	sort.Slice(files, func(i, j int) bool { return files[i].version < files[j].version })
	return files, nil
}

func loadAppliedVersions(ctx context.Context, pool *pgxpool.Pool) (map[int64]bool, error) {
	rows, err := pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("load applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[int64]bool)
	for rows.Next() {
		var version int64
		if scanErr := rows.Scan(&version); scanErr != nil {
			return nil, fmt.Errorf("scan migration version: %w", scanErr)
		}
		applied[version] = true
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return applied, nil
}
