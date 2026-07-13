#!/usr/bin/env python3
"""Sync only TODAY's market_daily_stats from the post-close symbol snapshot."""
import sqlite3
from datetime import datetime
from pathlib import Path
from zoneinfo import ZoneInfo

DB = Path("/root/AHRM/data/market.db")
TEHRAN = ZoneInfo("Asia/Tehran")


def main() -> None:
    today = datetime.now(TEHRAN).strftime("%Y-%m-%d")
    db = sqlite3.connect(DB)
    cur = db.cursor()
    row = cur.execute(
        """
        SELECT
          SUM(CASE WHEN status = 'positive' THEN 1 ELSE 0 END),
          SUM(CASE WHEN status = 'negative' THEN 1 ELSE 0 END),
          COUNT(*)
        FROM market_symbol_snapshot
        WHERE snapshot_date = ?
        """,
        (today,),
    ).fetchone()
    if not row or row[2] == 0:
        db.close()
        return
    pos, neg, tot = row
    cur.execute(
        """
        INSERT INTO market_daily_stats (date, positive, negative, total)
        VALUES (?, ?, ?, ?)
        ON CONFLICT(date) DO UPDATE SET
          positive = excluded.positive,
          negative = excluded.negative,
          total = excluded.total
        """,
        (today, pos, neg, tot),
    )
    db.commit()
    db.close()


if __name__ == "__main__":
    main()
