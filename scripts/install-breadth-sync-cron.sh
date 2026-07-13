#!/usr/bin/env bash
# Install nightly breadth sync cron on AHRM server (188.240.196.9 only).
# Runs after the 20:00 backfill to restore snapshot-based daily stats.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET="/usr/local/bin/ahrm-sync-breadth-stats.py"
CRON_LINE="5 20 * * * ${TARGET} # ahrm: restore breadth stats from post-close snapshots"

install -m 755 "${SCRIPT_DIR}/sync-breadth-from-snapshot.py" "${TARGET}"

TMP="$(mktemp)"
crontab -l 2>/dev/null | grep -v "ahrm-sync-breadth-stats" | grep -v "ahrm: restore breadth" > "${TMP}" || true
echo "${CRON_LINE}" >> "${TMP}"
crontab "${TMP}"
rm -f "${TMP}"

echo "Installed ${TARGET}"
crontab -l | grep ahrm-sync-breadth-stats
