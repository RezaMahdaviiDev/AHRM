#!/usr/bin/env python3
"""Re-sync market_daily_stats from symbol snapshots (post-close source of truth)."""
import sqlite3
from pathlib import Path

DB = Path("/root/AHRM/data/market.db")


def main() -> None:
    db = sqlite3.connect(DB)
    cur = db.cursor()
    cur.execute(
        """
        SELECT snapshot_date,
          SUM(CASE WHEN status = 'positive' THEN 1 ELSE 0 END),
          SUM(CASE WHEN status = 'negative' THEN 1 ELSE 0 END),
          COUNT(*)
        FROM market_symbol_snapshot
        GROUP BY snapshot_date
        """
    )
    for date, pos, neg, tot in cur.fetchall():
        cur.execute(
            """
            INSERT INTO market_daily_stats (date, positive, negative, total)
            VALUES (?, ?, ?, ?)
            ON CONFLICT(date) DO UPDATE SET
              positive = excluded.positive,
              negative = excluded.negative,
              total = excluded.total
            """,
            (date, pos, neg, tot),
        )
    db.commit()
    db.close()


if __name__ == "__main__":
    main()
