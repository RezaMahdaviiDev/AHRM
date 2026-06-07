#!/usr/bin/env bash
set -euo pipefail
export PATH=/usr/local/go/bin:$PATH
cd /root/AHRM

echo "=== 1. Unit + integration tests ==="
go test ./... -count=1

echo "=== 2. Telegram bot getMe ==="
TOKEN=$(grep '^TELEGRAM_BOT_TOKEN=' .env | cut -d= -f2-)
curl -sf "https://api.telegram.org/bot${TOKEN}/getMe" | grep -q '"ok":true'
echo "Bot OK: @Navasangirirbot"

echo "=== 3. Try discover chat_id from getUpdates ==="
CHAT_ID=$(curl -s "https://api.telegram.org/bot${TOKEN}/getUpdates" | python3 -c "
import sys,json
d=json.load(sys.stdin)
for u in d.get('result',[]):
    m=u.get('message') or u.get('edited_message')
    if m and 'chat' in m:
        print(m['chat']['id']); break
" 2>/dev/null || true)
if [ -n "$CHAT_ID" ]; then
  sed -i "s/^TELEGRAM_CHAT_ID=.*/TELEGRAM_CHAT_ID=${CHAT_ID}/" .env
  echo "Found chat_id=$CHAT_ID — sending test message"
  go run ./cmd/telegram-test
else
  echo "No chat_id yet — send /start to @Navasangirirbot then re-run: go run ./cmd/telegram-test"
fi

echo "=== 4. Start server ==="
FREE_PORT=$(python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()')
export HTTP_ADDR=":${FREE_PORT}"
go run ./cmd/server > /tmp/ahrm-fulltest.log 2>&1 &
PID=$!
trap "kill $PID 2>/dev/null || true" EXIT

for i in $(seq 1 30); do
  curl -sf "http://localhost:${FREE_PORT}/health" >/dev/null && break
  sleep 1
done

echo "health:" $(curl -sf "http://localhost:${FREE_PORT}/health")
echo "ready:" $(curl -sf "http://localhost:${FREE_PORT}/ready")

for path in /dashboard /arbitrage /hv /market /matrix; do
  echo "=== GET ${path} (may take up to 90s on first load) ==="
  code=$(curl -sf -o /tmp/page.html -w "%{http_code}" --max-time 120 "http://localhost:${FREE_PORT}${path}" || echo "000")
  echo "HTTP $code size=$(wc -c < /tmp/page.html 2>/dev/null || echo 0)"
  head -3 /tmp/page.html 2>/dev/null || true
done

echo "=== Done ==="
