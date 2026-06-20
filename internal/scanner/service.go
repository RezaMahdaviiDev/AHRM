package scanner

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"ahrm/internal/alerts"
	"ahrm/internal/arbitrage"
	"ahrm/internal/bullspread"
	"ahrm/internal/config"
	"ahrm/internal/coveredcall"
	"ahrm/internal/domain"
	"ahrm/internal/hv"
	"ahrm/internal/indicators"
	"ahrm/internal/ivcalc"
	"ahrm/internal/market"
	"ahrm/internal/matrix"
	"ahrm/internal/matrixalerts"
	"ahrm/internal/pairs"
	"ahrm/internal/sourcearena"
)

const hvCandleLookbackDays = 180

type Service struct {
	cfg        *config.Config
	client     *sourcearena.Client
	marketStore market.DailyStore
	backfilling atomic.Bool
	pairEngine        *pairs.Engine
	arbEngine         *arbitrage.Engine
	coveredCallEngine *coveredcall.Engine
	ivEngine          *ivcalc.Engine
	hvEngine          *hv.Engine
	bullSpreadEngine  *bullspread.Engine
	breadth    *indicators.BreadthEngine
	advance    *indicators.AdvanceDeclineEngine
	matrix      *matrix.Engine
	matrixRules []matrixalerts.Rule
	alerts      *alerts.Engine
}

type HVFetch struct {
	Symbol     string    `json:"symbol"`
	From       time.Time `json:"from"`
	To         time.Time `json:"to"`
	Resolution string    `json:"resolution"`
	Type       int       `json:"type"`
	TypeLabel  string    `json:"type_label"`
}

type Snapshot struct {
	GeneratedAt   time.Time                    `json:"generated_at"`
	Underlying    sourcearena.SymbolQuote      `json:"underlying"`
	HV            hv.Result                    `json:"hv"`
	HVFetch       HVFetch                      `json:"hv_fetch"`
	Breadth       indicators.IndicatorResult   `json:"breadth"`
	AdvanceDecline indicators.IndicatorResult  `json:"advance_decline"`
	Opportunities     []arbitrage.Opportunity   `json:"opportunities"`
	CoveredCalls      []coveredcall.CoveredCall `json:"covered_calls"`
	ImpliedVolatility []ivcalc.IVResult         `json:"implied_volatility"`
	CallMatrices      []matrix.Matrix           `json:"call_matrices"`
	PutMatrices       []matrix.Matrix           `json:"put_matrices"`
	BullSpreadsATM    []bullspread.Spread            `json:"bull_spreads_atm"`
	BullSpreadsOTM    []bullspread.Spread            `json:"bull_spreads_otm"`
	PriceChart        []sourcearena.Candle           `json:"price_chart"`
	Indicators        *sourcearena.TechnicalIndicators `json:"indicators,omitempty"`
	BackfillInProgress bool                          `json:"backfill_in_progress,omitempty"`
	Errors            []string                       `json:"errors,omitempty"`
}

func NewService(cfg *config.Config, client *sourcearena.Client, marketStore market.DailyStore, alertEngine *alerts.Engine) *Service {
	matrixRules, _ := matrixalerts.LoadRules(cfg.MatrixAlertsFile)
	return &Service{
		cfg:         cfg,
		client:      client,
		marketStore: marketStore,
		pairEngine:        pairs.NewEngine(),
		arbEngine:         arbitrage.NewEngine(),
		coveredCallEngine: coveredcall.NewEngine(),
		ivEngine:          ivcalc.NewEngine(),
		hvEngine:          hv.NewEngine(),
		breadth: indicators.NewBreadthEngine(indicators.Thresholds{
			High: cfg.Alerts.BreadthHighThreshold,
			Low:  cfg.Alerts.BreadthLowThreshold,
		}),
		advance: indicators.NewAdvanceDeclineEngine(indicators.Thresholds{
			High: cfg.Alerts.AdvanceHighThreshold,
			Low:  cfg.Alerts.AdvanceLowThreshold,
		}),
		bullSpreadEngine: bullspread.NewEngine(),
		matrix:      matrix.NewEngine(),
		matrixRules: matrixRules,
		alerts:      alertEngine,
	}
}

func (s *Service) Refresh(ctx context.Context) (Snapshot, error) {
	snap := Snapshot{GeneratedAt: time.Now().UTC(), BackfillInProgress: s.backfilling.Load()}
	if s.client == nil {
		snap.Errors = append(snap.Errors, "sourcearena client not configured")
		return snap, nil
	}

	options, err := s.client.FetchOptions(ctx)
	if err != nil {
		snap.Errors = append(snap.Errors, fmt.Sprintf("options: %v", err))
	}
	symbols, err := s.client.FetchAllSymbols(ctx)
	if err != nil {
		snap.Errors = append(snap.Errors, fmt.Sprintf("symbols: %v", err))
	}
	underlying, err := s.client.FetchSymbol(ctx, domain.UnderlyingSymbol)
	if err != nil {
		snap.Errors = append(snap.Errors, fmt.Sprintf("underlying: %v", err))
	} else {
		snap.Underlying = underlying
	}

	if s.marketStore != nil && len(symbols) > 0 {
		today := market.ClassifyDay(symbols)
		_ = s.marketStore.UpsertToday(ctx, today)
	}
	var history []indicators.DailyMarket
	if s.marketStore != nil {
		var histErr error
		history, histErr = s.marketStore.LastDays(ctx, 10)
		if histErr != nil {
			snap.Errors = append(snap.Errors, fmt.Sprintf("market history: %v", histErr))
		}
	}
	if len(history) > 0 {
		if breadth, bErr := s.breadth.Evaluate(history); bErr == nil {
			snap.Breadth = breadth
		} else {
			snap.Errors = append(snap.Errors, fmt.Sprintf("breadth: %v", bErr))
		}
		if ad, aErr := s.advance.Evaluate(history); aErr == nil {
			snap.AdvanceDecline = ad
		} else {
			snap.Errors = append(snap.Errors, fmt.Sprintf("advance_decline: %v", aErr))
		}
	}

	if !snap.BackfillInProgress {
		from := time.Now().AddDate(0, 0, -hvCandleLookbackDays)
		to := time.Now()
		hvReq := sourcearena.CandleRequest{
			Symbol:     domain.UnderlyingSymbol,
			From:       from,
			To:         to,
			Resolution: sourcearena.Resolution1D,
			Type:       sourcearena.AdjustCapAndDividend,
		}
		snap.HVFetch = HVFetch{
			Symbol:     hvReq.Symbol,
			From:       hvReq.From,
			To:         hvReq.To,
			Resolution: hvReq.Resolution,
			Type:       hvReq.Type,
			TypeLabel:  "افزایش سرمایه و سود نقدی",
		}
		candles, err := s.client.FetchCandles(ctx, hvReq)
		if err != nil {
			snap.Errors = append(snap.Errors, fmt.Sprintf("candles: %v", err))
		} else {
			if hvResult, hvErr := s.hvEngine.Calculate(candles); hvErr == nil {
				snap.HV = hvResult
			} else {
				snap.Errors = append(snap.Errors, fmt.Sprintf("hv: %v", hvErr))
			}
			if len(candles) > 0 {
				start := len(candles) - 30
				if start < 0 {
					start = 0
				}
				snap.PriceChart = candles[start:]
			}
		}
		if indResult, indErr := s.client.FetchIndicators(ctx, domain.UnderlyingSymbol); indErr == nil {
			snap.Indicators = indResult
		} else {
			snap.Errors = append(snap.Errors, fmt.Sprintf("indicators: %v", indErr))
		}
	}

	if len(options) > 0 {
		if snap.Underlying.ClosePrice > 0 {
			covered, ccErr := s.coveredCallEngine.CalculateAll(options, snap.Underlying.ClosePrice)
			if ccErr != nil {
				snap.Errors = append(snap.Errors, fmt.Sprintf("covered_call: %v", ccErr))
			} else {
				snap.CoveredCalls = covered
			}
			ivResults, ivErrs := s.ivEngine.CalculateAll(options, snap.Underlying.ClosePrice, s.cfg.RiskFreeRate)
			snap.ImpliedVolatility = ivResults
			for _, msg := range ivErrs {
				snap.Errors = append(snap.Errors, msg)
			}
		}
		if matched, pErr := s.pairEngine.Match(options); pErr == nil {
			opps, _ := s.arbEngine.CalculateAll(matched, snap.Underlying.ClosePrice)
			snap.Opportunities = opps
			for _, opp := range opps {
				if s.alerts != nil {
					_, _ = s.alerts.MaybeSendArbitrage(ctx, alerts.ArbitrageAlertInput{
						Symbol: opp.Symbol, Expiry: opp.Expiry, Strike: opp.Strike, ReturnPct: opp.ReturnPct,
					})
					_, _ = s.alerts.MaybeSendArbitrageR12Bale(ctx, alerts.ArbitrageAlertInput{
						Symbol: opp.Symbol, Expiry: opp.Expiry, Strike: opp.Strike, ReturnPct: opp.ReturnPct12_5,
					})
				}
			}
		} else {
			snap.Errors = append(snap.Errors, fmt.Sprintf("pairs: %v", pErr))
		}
		if callMatrices, err := s.matrix.BuildCalls(options); err == nil {
			snap.CallMatrices = callMatrices
		}
		if putMatrices, err := s.matrix.BuildPuts(options); err == nil {
			snap.PutMatrices = putMatrices
		}
		if snap.Underlying.ClosePrice > 0 {
			snap.BullSpreadsATM = s.bullSpreadEngine.CalculateAll(options, snap.Underlying.ClosePrice, bullspread.ATM)
			snap.BullSpreadsOTM = s.bullSpreadEngine.CalculateAll(options, snap.Underlying.ClosePrice, bullspread.OTM)
		}

		prices := make(map[string]float64, len(options))
		for _, opt := range options {
			prices[opt.Name] = opt.ClosePrice
		}
		for _, rule := range s.matrixRules {
			priceA, okA := prices[rule.SymbolA]
			priceB, okB := prices[rule.SymbolB]
			if !okA || !okB {
				continue
			}
			diff, triggered := rule.Evaluate(priceA, priceB)
			if triggered && s.alerts != nil {
				_, _ = s.alerts.MaybeSendMatrixAlert(ctx, rule.ID, diff, rule.Message)
			}
		}
	}

	if s.alerts != nil {
		if snap.Breadth.AlertState != "" {
			_, _ = s.alerts.MaybeSendBreadth(ctx, snap.Breadth.Average10Day, snap.Breadth.AlertState)
		}
		if snap.AdvanceDecline.AlertState != "" {
			_, _ = s.alerts.MaybeSendAdvanceDecline(ctx, snap.AdvanceDecline.Average10Day, snap.AdvanceDecline.AlertState)
		}
	}
	return snap, nil
}

// StartBackfillScheduler runs market history backfill on startup and then once per day
// at 20:00 Tehran time. Runs in a background goroutine; returns immediately.
func (s *Service) StartBackfillScheduler(ctx context.Context) {
	go func() {
		s.runBackfill(ctx)
		for {
			next := nextTehranTime(20, 0)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Until(next)):
				s.runBackfill(ctx)
			}
		}
	}()
}

func (s *Service) runBackfill(ctx context.Context) {
	if s.client == nil || s.marketStore == nil {
		return
	}
	s.backfilling.Store(true)
	defer s.backfilling.Store(false)

	bfCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	symbols, err := s.client.FetchAllSymbols(bfCtx)
	if err != nil || len(symbols) == 0 {
		return
	}
	_ = market.BackfillHistory(bfCtx, s.client, symbols, s.marketStore)
}

func nextTehranTime(hour, minute int) time.Time {
	loc, err := time.LoadLocation("Asia/Tehran")
	if err != nil {
		loc = time.FixedZone("IRST", 3*3600+30*60)
	}
	now := time.Now().In(loc)
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}
