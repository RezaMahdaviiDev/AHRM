CREATE TABLE IF NOT EXISTS market_daily_stats (
    day DATE PRIMARY KEY,
    positive INT NOT NULL,
    negative INT NOT NULL,
    total INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
