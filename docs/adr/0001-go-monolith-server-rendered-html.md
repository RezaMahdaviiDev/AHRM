# Single Go monolith serving server-rendered HTML

AHRM is one Go binary (`cmd/server`) that scans the market on a timer and serves
server-rendered HTML pages (`internal/server/templates`) plus `/health` and `/ready`.
We deliberately have no separate API service or JavaScript SPA: the audience is a small
number of operators, the data is read-mostly, and a single process keeps deployment
(one `systemd` unit) and the scan→render→alert loop trivial to reason about.
