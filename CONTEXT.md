# AHRM

Ubiquitous language for the AHRM options-arbitrage scanner: a Go service that watches
the Iranian `اهرم` options chain, computes arbitrage / volatility / breadth signals, and
sends Bale alerts. This file is a **glossary only** — definitions of domain
terms, not implementation. Read it before working in this repo; update it whenever a term
is added, renamed, or sharpened.

## Instruments

**Underlying**:
The single equity the whole scanner tracks, the Persian symbol `اهرم`. Every option's
basis is this symbol.
_Avoid_: stock, asset, AHRM (AHRM is the product name, not the instrument)

**Call**:
A call option on the Underlying; its symbol starts with the prefix `ضهرم`.
_Avoid_: ضهرم used for anything other than an AHRM call

**Put**:
A put option on the Underlying; its symbol starts with the prefix `طهرم`.
_Avoid_: طهرم used for anything other than an AHRM put

**Strike**:
The exercise price of an option.
_Avoid_: emal_price, exercise price, emal

**Expiry**:
An option's expiration date, always expressed in the Jalali (Persian) calendar as
`YYYY/MM/DD`.
_Avoid_: to_date, maturity, Gregorian expiry dates

**Days to Expiry**:
Whole **calendar** days from today until an option's Expiry.
_Avoid_: trading days to expiry (that phrase is reserved for the HV window)

**Pair**:
A Call and a Put on the Underlying sharing the **same Strike and same Expiry**. The unit
the Arbitrage engine scores.
_Avoid_: pair for bull-spread legs, or for matrix cells

## Arbitrage

**Opportunity**:
A scored Pair representing a conversion-style position, ranked by its Return.
_Avoid_: opportunity for covered calls, bull spreads, or matrix diffs (those are separate)

**Arbitrage Spread**:
The Call premium minus the Put premium of a Pair.
_Avoid_: bare "spread" (collides with Bull Spread) — always qualify it

**Capital**:
The net cash committed to an Opportunity's position.
_Avoid_: margin, notional

**Return (R)**:
The percentage return at Expiry of an Opportunity, given today's prices. The primary
arbitrage ranking metric.
_Avoid_: ReturnPct, return_pct, mixing with the bull-spread Reward-to-Risk

**Stressed Return (R12.5)**:
The same return as R but computed after stressing the Underlying up by 12.5%. Drives the
Bale arbitrage alert.
_Avoid_: R12, ReturnPct12_5, "R prime" without the +12.5% definition

## Volatility

**Historical Volatility (HV)**:
The annualised volatility of the Underlying's daily log returns over its most recent
**40 trading days**, shown as a percentage.
_Avoid_: realized vol; computing HV over calendar days; using option prices for HV

**Implied Volatility (IV)**:
The volatility implied by a Call's market price under Black–Scholes, shown as a
percentage. Computed for Calls only.
_Avoid_: IV on puts; reusing the HV multiplier for IV

**Risk-free Rate**:
The annual rate fed into Black–Scholes (Iranian risk-free proxy, default 20%).
_Avoid_: interest rate, discount rate

## Strategies

**Covered Call**:
Holding the Underlying while writing a Call against it, evaluated only for Calls with more
than 30 Days to Expiry.
_Avoid_: buy-write (use Covered Call)

**Net Cost**:
The Underlying price minus the Call premium received in a Covered Call. Also the Covered
Call's break-even Underlying price.
_Avoid_: cost basis; treating Net Cost and Break Even as different numbers

**Static ROI**:
A Covered Call's premium yield on Net Cost (its return if the Underlying is unchanged at
Expiry). Drives the Bale covered-call alert.
_Avoid_: the legacy `(S / NetCost) × 100` definition — the live metric is premium-based

**Max ROI**:
A Covered Call's return if the Underlying finishes at or above the Strike at Expiry.
_Avoid_: best-case return, max profit

**Bull Spread**:
A bull call spread — long a lower-Strike Call (K1), short a higher-Strike Call (K2), same
Expiry — surfaced in ATM and OTM variants.
_Avoid_: vertical spread, debit spread (use Bull Spread); reusing arbitrage terms here

**Reward-to-Risk (R)**:
A Bull Spread's max profit divided by its debit. Distinct from the arbitrage Return even
though both are written "R".
_Avoid_: writing it as plain "R" without "Bull Spread" context; sود/ریسک ambiguity

## Market Breadth

**Daily Market Stats**:
For one trading day, the count of traded symbols that closed Positive, Negative, or in
total, across the **Stock Universe** only (see below). Recorded only after 13:00 Tehran
time. A per-symbol snapshot (name, change %, status) is also persisted at the same time
and displayed as the Symbol Detail table on the `/market` page. The price basis for
classification is **قیمت پایانی** (weighted average closing price, `final_price_change_percent`
in the SourceArena API) — not قیمت آخرین معامله (`close_price_change_percent`).

**Stock Universe**:
The set of symbols included in breadth calculations: all symbols from the SourceArena
`/api/?all` feed whose `market` field is NOT in the non-stock blocklist. The following
`market` values are excluded — ETFs, commodity funds, and non-equity instruments:
`بازار صندوق های قابل معامله`, `صندوق های قابل معامله`, `صندوق های کالایی`,
`بازار ابزارهاي نوين مالي فرابورس`, `بازار نوآفرین - رشد`, `بازار نوآفرین - دانش بنیان`,
`بورس کالا`. Option symbols (any name containing a digit) are also excluded.
_Avoid_: "all traded symbols" — the universe is filtered; ~786 symbols, not ~1063
_Avoid_: advancers/decliners (those are derived views, not the raw counts); including
option symbols in the universe

**Positive / Negative**:
A symbol is Positive when its close changed by more than +0.5% on the day, Negative when
it changed by less than −0.5%. Moves within ±0.5% are neither.
_Avoid_: up/down, gainers/losers; assuming a 0% threshold

**Breadth Thrust**:
The share of the traded universe that is Positive on a day; the alerting signal is its
rolling 10-day average.
_Avoid_: breadth as advancers÷decliners (that is Advance/Decline)

**Advance/Decline**:
The ratio of Positive to Negative symbols on a day; the alerting signal is its rolling
10-day average.
_Avoid_: A/D line, breadth

## Aggregation & alerting

**Snapshot**:
The full bundle of scanner outputs produced by one refresh cycle (underlying quote, HV,
IV, breadth, opportunities, covered calls, spreads, matrices, chart, errors).
_Avoid_: tick, quote, or "response" for the whole bundle

**Matrix**:
A per-Expiry table of pairwise price differences among the Calls (or among the Puts) of
that Expiry.
_Avoid_: a single matrix mixing expiries or mixing calls and puts

**Matrix Alert**:
A user-configured rule that fires when the price difference between two named option
symbols crosses a threshold.
_Avoid_: automatic matrix alerts — only configured rules fire

**Alert**:
A signal pushed to Bale messenger. All alert types — arbitrage R, stressed arbitrage
(R12.5), covered-call Static ROI, breadth, advance/decline, and matrix alerts — are sent
to Bale. Sent alerts are recorded in `alert_history` in `data/alerts.db` (SQLite) to
suppress duplicates within a 24-hour window.
_Avoid_: Telegram (removed); Supabase/PostgreSQL (removed); assuming alerts are sent without persistence

**Dedup Key**:
The per-alert identity used to suppress duplicate Alerts for the same event.
_Avoid_: resending, idempotency key
