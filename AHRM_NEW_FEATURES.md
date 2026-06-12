# AHRM — New Feature Requirements (Client Requests)

This document specifies 5 features requested by the client. Implement them **one at a time**, in the order below, rebuilding and running tests after each one. Each section references the exact files/structs in the current codebase (`github.com/RezaMahdaviiDev/AHRM`).

---

## 1) Covered Call calculations

For **call options with more than 30 days to expiry** (reuse the existing 30-day filter logic from `internal/pairs/engine.go`, `MinDaysToExpiry = 30`, and `internal/jalali` for date parsing), calculate:

- `S` = underlying price (`Snapshot.Underlying.ClosePrice`, i.e. اهرم close price)
- `C` = call option close price (`Option.ClosePrice`)
- `K` = call option strike price (`Option.StrikePrice`, JSON field `emal_price`)
- `Net Cost = S - C`
- `Static ROI % = (S / Net Cost) * 100`
- `Max ROI % = ((K - Net Cost) / Net Cost) * 100`

Implement these **exactly as given** (even though Static ROI will always be close to 100% — this is the formula the client specified; do not "correct" it).

### Implementation
- New package `internal/coveredcall/engine.go`:
  ```go
  type CoveredCall struct {
      Symbol       string  `json:"symbol"`
      Expiry       string  `json:"expiry"`
      DaysToExpiry int     `json:"days_to_expiry"`
      Underlying   float64 `json:"underlying"` // S
      OptionPrice  float64 `json:"option_price"` // C
      Strike       float64 `json:"strike"` // K
      NetCost      float64 `json:"net_cost"`
      StaticROIPct float64 `json:"static_roi_pct"`
      MaxROIPct    float64 `json:"max_roi_pct"`
  }
  type Engine struct {
      Now     func() time.Time
      MinDays int // default 30, like pairs.MinDaysToExpiry
  }
  func NewEngine() *Engine
  func (e *Engine) CalculateAll(calls []sourcearena.Option, underlyingPrice float64) ([]CoveredCall, error)
  ```
  - Filter: only options where `domain.IsCallOption(opt.Name)` is true and days-to-expiry (via `jalali.ParseDate` + `jalali.CalendarDaysUntil`) is `> 30`.
  - Skip entries where `NetCost <= 0` (avoid division by zero / negative).
- In `internal/scanner/service.go`:
  - Add `coveredCallEngine *coveredcall.Engine` to `Service`, initialize in `NewService`.
  - Add `CoveredCalls []coveredcall.CoveredCall` to `Snapshot`.
  - In `Refresh()`, after options are fetched (and underlying price is available), call `s.coveredCallEngine.CalculateAll(options, snap.Underlying.ClosePrice)` and store in `snap.CoveredCalls`. Append errors to `snap.Errors` on failure (don't fail the whole refresh).
- New page `/covered-call`:
  - Register in `internal/server/pages.go`: `mux.HandleFunc("GET /covered-call", s.pageHandler("covered-call.html", "Covered Call"))`
  - New template `internal/server/templates/covered-call.html`, styled like `arbitrage.html` (same dark theme, same sortable-table JS pattern — copy the `<script>` sorting block).
  - Columns: نماد, سررسید, روز تا سررسید, قیمت سهم (S), قیمت آپشن (C), قیمت اعمال (K), هزینه خالص (Net Cost), Static ROI %, Max ROI %.
  - Add `<a href="/covered-call">کاوردکال</a>` to the header nav in **all** templates (dashboard, arbitrage, hv, market, matrix, covered-call) — same as the other nav links.

---

## 2) Per-expiry matrices + manual diff alerts

### 2a. Split matrices by expiry date

Currently `internal/matrix/engine.go` builds **one** matrix mixing all calls (and one for all puts) regardless of expiry, with `Cells[i][j] = prices[i] - prices[j]`.

Change this so that **each expiry date gets its own separate matrix**.

- Update `matrix.Matrix` struct: add `Expiry string `json:"expiry"`` field.
- Update `(e *Engine) build(...)`:
  - Group filtered options by `opt.ExpiryDate` (`to_date` field, e.g. `"1405/12/15"`).
  - For each expiry group, build a separate `Matrix{Kind: kind, Expiry: expiryDate, Symbols, Prices, Cells}` (same pairwise-diff logic as now, but only within that expiry's options).
  - Sort groups by expiry date ascending (use `internal/jalali` for correct chronological sort of Jalali date strings).
- Change `BuildCalls` / `BuildPuts` return type from `(Matrix, error)` to `([]Matrix, error)`.
- Update `internal/scanner/service.go`:
  - `Snapshot.CallMatrix matrix.Matrix` → `Snapshot.CallMatrices []matrix.Matrix`
  - `Snapshot.PutMatrix matrix.Matrix` → `Snapshot.PutMatrices []matrix.Matrix`
  - Update the calls to `s.matrix.BuildCalls(options)` / `BuildPuts(options)` accordingly.
- Update `internal/server/templates/matrix.html`:
  - Loop over `.Snapshot.CallMatrices` and `.Snapshot.PutMatrices`, rendering one heading + table per expiry, e.g. `<h2>Call Matrix — سررسید {{.Expiry}}</h2>` followed by the table for that matrix.
  - Update existing tests (`internal/matrix/engine_test.go`, `internal/scanner/service_test.go`, `internal/server/pages_test.go`) for the new return types.

### 2b. Manual price-diff alerts between two symbols

The client wants to manually configure rules like: "if the price difference between ضهرم4030 and ضهرم4031 reaches 1200, alert me".

- New config file `configs/matrix_alerts.json` (create the `configs/` directory), example:
  ```json
  [
    {
      "id": "rule1",
      "symbol_a": "ضهرم4030",
      "symbol_b": "ضهرم4031",
      "operator": ">=",
      "threshold": 1200,
      "message": "اختلاف ضهرم4030 و ضهرم4031 به 1200 رسید"
    }
  ]
  ```
  - `operator` supports `>=`, `<=`, `==` (diff is `price[symbol_a] - price[symbol_b]`).
- New package `internal/matrixalerts/`:
  - `type Rule struct { ID, SymbolA, SymbolB, Operator, Message string; Threshold float64 }`
  - `func LoadRules(path string) ([]Rule, error)` — reads JSON file; if file doesn't exist, return empty slice (no error) so this feature is fully optional.
  - `func (r Rule) Evaluate(priceA, priceB float64) (diff float64, triggered bool)`
- In `internal/config/config.go`: add `MatrixAlertsFile string` to `Config`, env var `MATRIX_ALERTS_FILE`, default `"configs/matrix_alerts.json"`.
- In `internal/alerts/engine.go`: add a new method:
  ```go
  func (e *Engine) MaybeSendMatrixAlert(ctx context.Context, ruleID string, diff float64, message string) (bool, error)
  ```
  Follow the exact same dedup pattern as `MaybeSendArbitrage` (key = `fmt.Sprintf("matrix:%s:%.0f", ruleID, diff)`, alertType = `"matrix"`).
- In `internal/scanner/service.go`:
  - Load rules once at startup (in `NewService`, store `[]matrixalerts.Rule` on `Service`).
  - In `Refresh()`, after options are fetched, build a `map[string]float64` of symbol → `ClosePrice` from all options (calls + puts). For each rule, look up both symbols; if both found, call `rule.Evaluate(priceA, priceB)`; if triggered and `s.alerts != nil`, call `s.alerts.MaybeSendMatrixAlert(ctx, rule.ID, diff, rule.Message)`.
  - If a rule's symbols aren't found in the current snapshot, skip silently (don't add to `snap.Errors`).

This is a config-file-based approach for now (no web UI for editing rules) — the client can edit `configs/matrix_alerts.json` directly, or we build a UI for it later if requested.

---

## 3) Advance/Decline thresholds → 0.6 / 1.4

Currently in `internal/config/config.go`:
```go
AdvanceHighThreshold: parseFloatEnv("ALERT_ADVANCE_HIGH", 2.0),
AdvanceLowThreshold:  parseFloatEnv("ALERT_ADVANCE_LOW", 0.8),
```

Change the **defaults** to:
```go
AdvanceHighThreshold: parseFloatEnv("ALERT_ADVANCE_HIGH", 1.4),
AdvanceLowThreshold:  parseFloatEnv("ALERT_ADVANCE_LOW", 0.6),
```

Also:
- Update `.env.example` (and `.env` if it has these keys) to set `ALERT_ADVANCE_HIGH=1.4` and `ALERT_ADVANCE_LOW=0.6` (or remove the lines so the new defaults apply).
- Update the hardcoded display text in `internal/server/templates/market.html`:
  ```html
  <p class="muted">آستانه بالا: 2.0 | آستانه پایین: 0.8</p>
  ```
  → change to `1.4` and `0.6`.
- No other logic changes needed — `indicators.AdvanceDeclineEngine` and the alert-state logic already work correctly with these thresholds (`AlertState`: `>= High` → "high", `<= Low` → "low").

---

## 4) Arbitrage page: replace bid/ask row-1 volumes with traded volume

Currently `arbitrage.Opportunity` (in `internal/arbitrage/engine.go`) has:
```go
SellRow1Volume float64 `json:"1_sell_volume"`
BuyRow1Volume  float64 `json:"1_buy_volume"`
```
populated from `Option.SellRow1Volume` / `Option.BuyRow1Volume` (JSON fields `1_sell_volume`, `1_buy_volume`).

The client wants these two columns **removed** and replaced with a single **"حجم معامله شده" (traded volume)** column.

### Steps
1. **Find the correct API field name.** The SourceArena options/symbols response includes a traded-volume field (we saw `"trade_volume"` in the symbol/underlying response during testing — confirm the exact field name appears the same way in the **options** (`?all=e`) response). Add it to `internal/sourcearena/models.go`:
   ```go
   // in optionWire
   TradeVolume flexFloat `json:"trade_volume"`
   // in Option
   TradeVolume float64 `json:"trade_volume"`
   ```
   and populate it in `wiresToOptions`.
   - **If you cannot verify the exact field name from the testdata/docs**, add a debug log (similar to the previous temporary debug logging pattern) that prints the raw JSON keys of the first options item, ask me to run it on Iranian internet, then finalize the field name and remove the debug log.

2. Update `arbitrage.Opportunity`:
   - Remove `SellRow1Volume` and `BuyRow1Volume` fields.
   - Add `TradeVolume float64 \`json:"trade_volume"\``.
   - In `Engine.Calculate()`, set `opp.TradeVolume = pair.Call.TradeVolume`.

3. Update `internal/server/templates/arbitrage.html`:
   - Remove the two `<th>` columns "حجم فروش ردیف اول" (data-col=1) and "حجم خرید ردیف اول" (data-col=2), and their corresponding `<td>` cells.
   - Add one new `<th>` "حجم معامله شده" with `data-type="number"`.
   - **Renumber all `data-col` attributes** on the remaining `<th>` elements sequentially (0, 1, 2, ...) since two columns were removed and one was added — the sort JS relies on these indices matching column positions exactly.

4. Update `internal/arbitrage/engine_test.go` and any other tests referencing the removed fields.

---

## 5) Implied Volatility (IV) via Black-Scholes

Add IV calculation for each call option (paired with its strike/expiry, same pairs used for arbitrage — i.e. `>30` days to expiry) on the `/hv` page.

### Black-Scholes formulas
- Inputs: `S` (underlying price), `K` (strike), `T` (time to expiry in years), `r` (risk-free rate), `σ` (volatility).
- `d1 = (ln(S/K) + (r + σ²/2) * T) / (σ * sqrt(T))`
- `d2 = d1 - σ * sqrt(T)`
- Call price `= S * N(d1) - K * e^(-r*T) * N(d2)`, where `N` = standard normal CDF (`0.5 * (1 + erf(x / sqrt(2)))`, use `math.Erf`).
- **Implied Volatility**: given the market call price, solve for `σ` such that the BS call price equals the market price. Use **bisection** (robust, no derivatives needed): search `σ` in range `[0.001, 5.0]` (i.e. 0.1% to 500%), ~100 iterations or until price difference `< 0.01`. Return an error if the market price is outside the theoretical bounds (e.g. `marketPrice <= max(0, S - K*e^(-r*T))` or `marketPrice >= S`).

### Implementation
- New package `internal/blackscholes/`:
  ```go
  func CallPrice(S, K, T, r, sigma float64) float64
  func ImpliedVolatility(marketPrice, S, K, T, r float64) (float64, error)
  ```
  with unit tests (e.g. compute CallPrice for a known sigma, then verify ImpliedVolatility recovers approximately the same sigma).

- Add risk-free rate to config: `internal/config/config.go` → `RiskFreeRate float64`, env var `RISK_FREE_RATE`, default `0.20` (20% — typical Iranian risk-free proxy; the client may adjust via `.env`).

- New struct, e.g. in a new package `internal/ivcalc/` or alongside `coveredcall`:
  ```go
  type IVResult struct {
      Symbol       string  `json:"symbol"`
      Strike       float64 `json:"strike"`
      Expiry       string  `json:"expiry"`
      DaysToExpiry int     `json:"days_to_expiry"`
      OptionPrice  float64 `json:"option_price"`
      Underlying   float64 `json:"underlying"`
      IVPct        float64 `json:"iv_pct"`
  }
  ```
  Engine that takes call options (>30 days to expiry, same filter as covered call), underlying price, and config risk-free rate; for each, compute `T = DaysToExpiry / 365.0` and call `blackscholes.ImpliedVolatility(option.ClosePrice, underlyingPrice, option.StrikePrice, T, riskFreeRate)`. Skip (don't error the whole batch) any option where IV calculation fails — log to `snap.Errors` instead.

- In `internal/scanner/service.go`:
  - Add `Snapshot.ImpliedVolatility []ivcalc.IVResult`.
  - Compute it in `Refresh()` alongside covered calls.

- Update `internal/server/templates/hv.html`:
  - Add a new `<div class="card">` section titled "نوسان ضمنی آپشن‌ها (IV)" with a table: نماد, سررسید, روز تا سررسید, قیمت اعمال (K), قیمت آپشن (C), IV %.
  - Reuse the existing sortable-table pattern if helpful (optional).

---

## General notes for all tasks
- After each task: run `go vet ./...` and `go test ./...`, then `go build -o bin/server.exe ./cmd/server`.
- Keep backward compatibility where reasonable, but it's OK to change `Snapshot` field names/types as specified — update all templates and tests that reference them.
- Do not touch the SourceArena auth logic (`internal/sourcearena/client.go`) except for adding the `trade_volume` field in task 4.
- For tasks involving live SourceArena data verification (task 4's field name, task 1/5 sanity-checking real numbers), prepare the change and ask me to test on Iranian internet — you cannot reach SourceArena directly.
