CREATE TABLE IF NOT EXISTS alert_history (
    id BIGSERIAL PRIMARY KEY,
    alert_type TEXT NOT NULL,
    alert_key TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (alert_type, alert_key)
);

CREATE INDEX IF NOT EXISTS idx_alert_history_sent_at ON alert_history (sent_at DESC);
