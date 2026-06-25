package scanner

import (
	"context"
	"fmt"
	"sync"
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

type BackfillProgress struct {
	CurrentBatch int
	TotalBatches int
	Symbols      int
	TotalSymbols int
}

type Service struct {
	cfg           *config.Config
	client        *sourcearena.Client
	marketStore   market.DailyStore
	symbolStore   market.SymbolSnapshotStore
	registryStore market.SymbolRegistryStore
	backfilling     atomic.Bool
	bfMu            sync.Mutex
	bfProgress      BackfillProgress
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

// DailyRow is a single day's market breadth data for display in the UI.
type DailyRow struct {
	Date        string  `json:"date"`
	Positive    int     `json:"positive"`
	Negative    int     `json:"negative"`
	Total       int     `json:"total"`
	PctPositive float64 `json:"pct_positive"`
	InWindow    bool    `json:"in_window"` // true = part of the 10-day MA window
}

type Snapshot struct {
	GeneratedAt   time.Time                    `json:"generated_at"`
	Underlying    sourcearena.SymbolQuote      `json:"underlying"`
	HV            hv.Result                    `json:"hv"`
	HVFetch       HVFetch                      `json:"hv_fetch"`
	Breadth       indicators.IndicatorResult   `json:"breadth"`
	AdvanceDecline indicators.IndicatorResult  `json:"advance_decline"`
	DailyHistory      []DailyRow                    `json:"daily_history,omitempty"`
	Opportunities     []arbitrage.Opportunity       `json:"opportunities"`
	CoveredCalls      []coveredcall.CoveredCall     `json:"covered_calls"`
	ImpliedVolatility []ivcalc.IVResult             `json:"implied_volatility"`
	CallMatrices      []matrix.Matrix               `json:"call_matrices"`
	PutMatrices       []matrix.Matrix               `json:"put_matrices"`
	BullSpreadsATM    []bullspread.Spread            `json:"bull_spreads_atm"`
	BullSpreadsOTM    []bullspread.Spread            `json:"bull_spreads_otm"`
	PriceChart        []sourcearena.Candle           `json:"price_chart"`
	Indicators        *sourcearena.TechnicalIndicators `json:"indicators,omitempty"`
	SymbolRows        []indicators.SymbolRow         `json:"symbol_rows,omitempty"`
	QueueCandidates   []market.QueueCandidate        `json:"queue_candidates,omitempty"`
	BackfillInProgress bool                          `json:"backfill_in_progress,omitempty"`
	BackfillProgress   BackfillProgress              `json:"backfill_progress,omitempty"`
	Errors            []string                       `json:"errors,omitempty"`
}

func NewService(cfg *config.Config, client *sourcearena.Client, marketStore market.DailyStore, symbolStore market.SymbolSnapshotStore, registryStore market.SymbolRegistryStore, alertEngine *alerts.Engine) *Service {
	matrixRules, _ := matrixalerts.LoadRules(cfg.MatrixAlertsFile)
	return &Service{
		cfg:           cfg,
		client:        client,
		marketStore:   marketStore,
		symbolStore:   symbolStore,
		registryStore: registryStore,
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
	s.bfMu.Lock()
	bfProg := s.bfProgress
	s.bfMu.Unlock()
	snap := Snapshot{GeneratedAt: time.Now().UTC(), BackfillInProgress: s.backfilling.Load(), BackfillProgress: bfProg}
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

	// Scan for buy-queue (صف خرید) candidates suitable for سرخطی tomorrow.
	if len(symbols) > 0 {
		names := make([]string, 0, len(symbols))
		for _, sym := range symbols {
			names = append(names, sym.Name)
		}
		var newSyms []string
		var streaks map[string]int
		if s.registryStore != nil {
			newSyms, _ = s.registryStore.RegisterSymbols(ctx, names)
		}
		newSymsSet := make(map[string]bool, len(newSyms))
		for _, n := range newSyms {
			newSymsSet[n] = true
		}
		candidates := market.ScanQueue(symbols, newSymsSet, nil)
		// enrich with streak data after we know which symbols qualified
		if s.registryStore != nil && len(candidates) > 0 {
			queueNames := make([]string, len(candidates))
			for i, c := range candidates {
				queueNames[i] = c.Name
			}
			streaks, _ = s.registryStore.UpsertQueueStreaks(ctx, queueNames)
			for i := range candidates {
				candidates[i].StreakDays = streaks[candidates[i].Name]
			}
		}
		snap.QueueCandidates = candidates
	}

	// Record today's breadth snapshot after 13:00 Tehran time (post-session data).
	if isTehranAfter(13, 0) && s.marketStore != nil && len(symbols) > 0 {
		today := market.ClassifyDay(symbols)
		_ = s.marketStore.UpsertToday(ctx, today)
		if s.symbolStore != nil {
			symRows := market.SymbolRows(symbols)
			date := time.Now().UTC().Format("2006-01-02")
			_ = s.symbolStore.UpsertSymbolSnapshot(ctx, date, symRows)
		}
	}
	var history []indicators.DailyMarket
	if s.marketStore != nil {
		var histErr error
		history, histErr = s.marketStore.LastDays(ctx, 30)
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
		rows := make([]DailyRow, 0, len(history))
		windowStart := len(history) - 10
		if windowStart < 0 {
			windowStart = 0
		}
		for i, d := range history {
			var pct float64
			if d.Total > 0 {
				pct = float64(d.Positive) / float64(d.Total) * 100
			}
			rows = append(rows, DailyRow{
				Date:        d.Date,
				Positive:    d.Positive,
				Negative:    d.Negative,
				Total:       d.Total,
				PctPositive: pct,
				InWindow:    i >= windowStart,
			})
		}
		// reverse for newest-first display
		for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
			rows[i], rows[j] = rows[j], rows[i]
		}
		snap.DailyHistory = rows
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
				for _, cc := range covered {
					if s.alerts != nil {
						_, _ = s.alerts.MaybeSendCoveredCallROI(ctx, alerts.CoveredCallAlertInput{
							Symbol:       cc.Symbol,
							Expiry:       cc.Expiry,
							Strike:       cc.Strike,
							StaticROIPct: cc.StaticROIPct,
						})
					}
				}
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
					_, _ = s.alerts.MaybeSendArbitrageR12(ctx, alerts.ArbitrageAlertInput{
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

	if s.symbolStore != nil {
		if _, symRows, err := s.symbolStore.LatestSymbolSnapshot(ctx); err == nil {
			snap.SymbolRows = symRows
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
	// Only set the flag when we actually need to backfill — prevents the banner
	// from appearing on normal restarts where data is already sufficient.
	if need, _ := market.NeedsBackfill(ctx, s.marketStore); !need {
		return
	}
	s.backfilling.Store(true)
	defer s.backfilling.Store(false)

	bfCtx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()
	symbols, err := s.client.FetchAllSymbols(bfCtx)
	if err != nil || len(symbols) == 0 {
		return
	}
	onProgress := func(batch, totalBatches, symbolsDone, totalSymbols int) {
		s.bfMu.Lock()
		s.bfProgress = BackfillProgress{
			CurrentBatch: batch,
			TotalBatches: totalBatches,
			Symbols:      symbolsDone,
			TotalSymbols: totalSymbols,
		}
		s.bfMu.Unlock()
	}
	_ = market.BackfillHistory(bfCtx, s.client, symbols, s.marketStore, onProgress)
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

func isTehranAfter(hour, minute int) bool {
	loc, err := time.LoadLocation("Asia/Tehran")
	if err != nil {
		loc = time.FixedZone("IRST", 3*3600+30*60)
	}
	now := time.Now().In(loc)
	return now.Hour() > hour || (now.Hour() == hour && now.Minute() >= minute)
}
