package scanner

import (
	"context"
	"fmt"
	"time"

	"ahrm/internal/alerts"
	"ahrm/internal/arbitrage"
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
	marketStore *market.Store
	pairEngine        *pairs.Engine
	arbEngine         *arbitrage.Engine
	coveredCallEngine *coveredcall.Engine
	ivEngine          *ivcalc.Engine
	hvEngine          *hv.Engine
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
	Opportunities []arbitrage.Opportunity      `json:"opportunities"`
	CoveredCalls      []coveredcall.CoveredCall    `json:"covered_calls"`
	ImpliedVolatility []ivcalc.IVResult            `json:"implied_volatility"`
	CallMatrices      []matrix.Matrix              `json:"call_matrices"`
	PutMatrices   []matrix.Matrix              `json:"put_matrices"`
	Errors        []string                     `json:"errors,omitempty"`
}

func NewService(cfg *config.Config, client *sourcearena.Client, marketStore *market.Store, alertEngine *alerts.Engine) *Service {
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
		matrix:      matrix.NewEngine(),
		matrixRules: matrixRules,
		alerts:      alertEngine,
	}
}

func (s *Service) Refresh(ctx context.Context) (Snapshot, error) {
	snap := Snapshot{GeneratedAt: time.Now().UTC()}
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

	if len(symbols) > 0 {
		today := market.ClassifyDay(symbols)
		_ = s.marketStore.UpsertToday(ctx, today)
		history, histErr := s.marketStore.LastDays(ctx, 10)
		if histErr != nil {
			snap.Errors = append(snap.Errors, fmt.Sprintf("market history: %v", histErr))
		}
		if len(history) == 0 {
			history = []indicators.DailyMarket{today}
		}
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
	} else if hvResult, hvErr := s.hvEngine.Calculate(candles); hvErr == nil {
		snap.HV = hvResult
	} else {
		snap.Errors = append(snap.Errors, fmt.Sprintf("hv: %v", hvErr))
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
						Expiry: opp.Expiry, Strike: opp.Strike, ReturnPct: opp.ReturnPct,
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
