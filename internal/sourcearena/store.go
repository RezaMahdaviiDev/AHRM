package sourcearena

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRawStore struct {
	pool *pgxpool.Pool
}

func NewPostgresRawStore(pool *pgxpool.Pool) *PostgresRawStore {
	return &PostgresRawStore{pool: pool}
}

func (s *PostgresRawStore) SaveRaw(ctx context.Context, endpoint string, statusCode int, body []byte) error {
	if s == nil || s.pool == nil {
		return nil
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO api_raw_responses (endpoint, status_code, body) VALUES ($1, $2, $3::jsonb)`,
		endpoint, statusCode, string(body),
	)
	return err
}

type NopRawStore struct{}

func (NopRawStore) SaveRaw(context.Context, string, int, []byte) error { return nil }
