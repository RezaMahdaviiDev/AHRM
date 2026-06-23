# HV requests a 180-calendar-day candle window

Historical Volatility needs 41 daily closes (40 log returns). We request a **180
calendar-day** candle window rather than something closer to 40, because Iranian market
weekends and holidays mean a tighter window (e.g. 90 days) does not reliably yield 41
trading-day candles. 180 days is the empirically safe margin (`hvCandleLookbackDays` in
`internal/scanner/service.go`); see `SOURCEARENA_API.md`.
