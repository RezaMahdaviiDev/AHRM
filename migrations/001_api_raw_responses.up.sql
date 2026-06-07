CREATE TABLE IF NOT EXISTS api_raw_responses (
    id BIGSERIAL PRIMARY KEY,
    endpoint TEXT NOT NULL,
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status_code INT NOT NULL,
    body JSONB NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_api_raw_responses_endpoint_fetched
    ON api_raw_responses (endpoint, fetched_at DESC);
