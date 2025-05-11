package strategies

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/markcheno/go-talib"
	"github.com/raykavin/backnrun/bot"
	"github.com/raykavin/backnrun/core"
	"github.com/raykavin/backnrun/indicator"
)

// TrendMasterConfig contains all configurable parameters for the strategy
type TrendMasterConfig struct {
	// Base timeframe for the strategy
	Timeframe string

	// Higher timeframe for trend filtering
	HigherTimeframe string

	// Warmup period for historical calculations
	WarmupPeriod int

	// Maximum trades per day
	MaxTradesPerDay int

	// Trading hours control
	TradingHoursEnabled bool
	TradingStartHour    int
	TradingEndHour      int

	// EMA indicator parameters
	EmaFastPeriod int
	EmaSlowPeriod int
	EmaLongPeriod int

	// MACD parameters
	MacdFastPeriod   int
	MacdSlowPeriod   int
	MacdSignalPeriod int

	// ADX parameters
	AdxPeriod          int
	AdxThreshold       float64
	AdxMinimumDiSpread float64

	// RSI parameters
	UseRsiFilter         bool
	RsiPeriod            int
	RsiOverbought        float64
	RsiOversold          float64
	RsiExtremeOverbought float64

	// ATR parameters
	AtrPeriod           int
	AtrMultiplier       float64
	VolatilityThreshold float64

	// Volume parameters
	UseVolFilter bool
	VolAvgPeriod int
	VolMinRatio  float64

	// Entry control
	UseHigherTfConfirmation bool

	// Sentiment filter
	UseSentimentFilter bool
	SentimentThreshold float64

	// Market correlation
	UseMarketCorrelation         bool
	CorrelationReferenceSymbol   string
	CorrelationPeriod            int
	NegativeCorrelationThreshold float64

	// Position management
	PositionSize        float64
	MaxRiskPerTrade     float64
	TrailingStopPercent float64

	// Adaptive position sizing
	UseAdaptiveSize       bool
	WinIncreaseFactor     float64
	LossReductionFactor   float64
	MinPositionSizeFactor float64
	MaxPositionSizeFactor float64

	// Partial take profit
	UsePartialTakeProfit bool
	PartialExitLevels    []PartialExitLevel

	// Dynamic targets
	UseDynamicTargets bool
	BaseTarget        float64
	AtrTargetFactor   float64
	MinTarget         float64
	MaxTarget         float64

	// Quick exits
	UseMacdReversalExit   bool
	MacdReversalThreshold float64
	UseAdxFallingExit     bool
	UsePriceActionExit    bool

	// Market-specific settings
	MarketSpecificSettings map[string]MarketSpecificConfig
}

// PartialExitLevel defines a partial exit level
type PartialExitLevel struct {
	Percentage   float64
	Target       float64
	TrailingOnly bool
}

// MarketSpecificConfig contains settings specific to each market type
type MarketSpecificConfig struct {
	VolatilityThreshold float64
	TrailingStopPercent float64
	AtrMultiplier       float64
}

// PartialPosition stores details of a partial position
type PartialPosition struct {
	Quantity   float64
	EntryPrice float64
	OrderID    int64
	Level      int
}

// TrendMaster implements a strategy that combines multiple indicators
// to identify strong trends and generate entry and exit signals
type TrendMaster struct {
	// Strategy configuration
	config TrendMasterConfig

	// Internal state
	marketType             map[string]string // "crypto", "forex", "stocks"
	higherTfCache          map[string]*core.Dataframe
	higherTfLastUpdate     map[string]time.Time
	marketSentiment        map[string]float64
	correlationValues      map[string][]float64
	correlationRef         map[string][]float64
	marketCorrelation      map[string]float64
	winCount               int
	lossCount              int
	winStreak              int
	lossStreak             int
	consecutiveLosses      int
	dailyTradeCount        map[string]int
	lastTradeDate          string
	lastTradeResult        map[string]bool
	partialPositions       map[string][]PartialPosition
	partialOrders          map[string][]int64
	activeOrders           map[string]map[int]int64
	lastPrice              map[string]float64
	entryPrice             map[string]float64
	positionSize           map[string]float64
	isDataFrameInitialized map[string]bool
}

// NewTrendMaster creates a new instance of the strategy with defined parameters
func NewTrendMaster(config TrendMasterConfig) *TrendMaster {
	return &TrendMaster{
		config:                 config,
		marketType:             make(map[string]string),
		higherTfCache:          make(map[string]*core.Dataframe),
		higherTfLastUpdate:     make(map[string]time.Time),
		marketSentiment:        make(map[string]float64),
		correlationValues:      make(map[string][]float64),
		correlationRef:         make(map[string][]float64),
		marketCorrelation:      make(map[string]float64),
		dailyTradeCount:        make(map[string]int),
		lastTradeResult:        make(map[string]bool),
		partialPositions:       make(map[string][]PartialPosition),
		partialOrders:          make(map[string][]int64),
		activeOrders:           make(map[string]map[int]int64),
		lastPrice:              make(map[string]float64),
		entryPrice:             make(map[string]float64),
		positionSize:           make(map[string]float64),
		isDataFrameInitialized: make(map[string]bool),
	}
}

// Timeframe returns the required timeframe for this strategy
func (t TrendMaster) Timeframe() string {
	return t.config.Timeframe
}

// WarmupPeriod returns the number of candles needed before the strategy is ready
func (t TrendMaster) WarmupPeriod() int {
	return t.config.WarmupPeriod
}

// Indicators calculates and returns the indicators used by this strategy
func (t TrendMaster) Indicators(df *core.Dataframe) []core.ChartIndicator {
	t.initializeDataFrame(df)
	t.calculateIndicators(df)
	t.updateHigherTimeframeCache(df)
	t.updateCorrelationData(df)
	t.detectMarketType(df.Pair)

	return t.createChartIndicators(df)
}

// initializeDataFrame initializes dataframe-specific data structures if needed
func (t *TrendMaster) initializeDataFrame(df *core.Dataframe) {
	if !t.isDataFrameInitialized[df.Pair] {
		t.isDataFrameInitialized[df.Pair] = true

		// Initialize arrays for correlation
		t.correlationValues[df.Pair] = make([]float64, 0, t.config.CorrelationPeriod*2)

		// If we're using correlation and this is the reference pair
		if t.config.UseMarketCorrelation && strings.Contains(df.Pair, t.config.CorrelationReferenceSymbol) {
			t.correlationRef[df.Pair] = make([]float64, 0, t.config.CorrelationPeriod*2)
		}
	}
}

// calculateIndicators computes all technical indicators for the dataframe
func (t *TrendMaster) calculateIndicators(df *core.Dataframe) {
	// Calculate EMAs
	df.Metadata["ema_fast"] = indicator.EMA(df.Close, t.config.EmaFastPeriod)
	df.Metadata["ema_slow"] = indicator.EMA(df.Close, t.config.EmaSlowPeriod)
	df.Metadata["ema_long_high"] = indicator.EMA(df.High, t.config.EmaLongPeriod)
	df.Metadata["ema_long_low"] = indicator.EMA(df.Low, t.config.EmaLongPeriod)

	// Calculate MACD
	df.Metadata["macd"], df.Metadata["macd_signal"], df.Metadata["macd_hist"] = indicator.MACD(
		df.Close,
		t.config.MacdFastPeriod,
		t.config.MacdSlowPeriod,
		t.config.MacdSignalPeriod,
	)

	// Calculate ADX and directional indicators
	df.Metadata["adx"] = indicator.ADX(df.High, df.Low, df.Close, t.config.AdxPeriod)
	df.Metadata["plus_di"] = talib.PlusDI(df.High, df.Low, df.Close, t.config.AdxPeriod)
	df.Metadata["minus_di"] = talib.MinusDI(df.High, df.Low, df.Close, t.config.AdxPeriod)

	// Add ATR for volatility calculation
	df.Metadata["atr"] = talib.Atr(df.High, df.Low, df.Close, t.config.AtrPeriod)

	// Calculate RSI for additional filtering
	if t.config.UseRsiFilter {
		df.Metadata["rsi"] = talib.Rsi(df.Close, t.config.RsiPeriod)
	}

	// Calculate volume average for filtering
	if t.config.UseVolFilter {
		df.Metadata["vol_avg"] = indicator.SMA(df.Volume, t.config.VolAvgPeriod)
	}
}

// createChartIndicators returns indicators for visualization
func (t *TrendMaster) createChartIndicators(df *core.Dataframe) []core.ChartIndicator {
	return []core.ChartIndicator{
		{
			Overlay:   true,
			GroupName: "Moving Averages",
			Time:      df.Time,
			Metrics: []core.IndicatorMetric{
				{
					Values: df.Metadata["ema_fast"],
					Name:   "EMA " + string(rune(t.config.EmaFastPeriod)),
					Color:  "blue",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["ema_slow"],
					Name:   "EMA " + string(rune(t.config.EmaSlowPeriod)),
					Color:  "green",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["ema_long_high"],
					Name:   "EMA " + string(rune(t.config.EmaLongPeriod)) + " (High)",
					Color:  "purple",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["ema_long_low"],
					Name:   "EMA " + string(rune(t.config.EmaLongPeriod)) + " (Low)",
					Color:  "orange",
					Style:  core.StyleLine,
				},
			},
		},
		{
			Overlay:   false,
			GroupName: "MACD",
			Time:      df.Time,
			Metrics: []core.IndicatorMetric{
				{
					Values: df.Metadata["macd"],
					Name:   "MACD Line",
					Color:  "blue",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["macd_signal"],
					Name:   "Signal Line",
					Color:  "red",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["macd_hist"],
					Name:   "Histogram",
					Color:  "green",
					Style:  core.StyleHistogram,
				},
			},
		},
		{
			Overlay:   false,
			GroupName: "ADX",
			Time:      df.Time,
			Metrics: []core.IndicatorMetric{
				{
					Values: df.Metadata["adx"],
					Name:   "ADX",
					Color:  "black",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["plus_di"],
					Name:   "+DI",
					Color:  "green",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["minus_di"],
					Name:   "-DI",
					Color:  "red",
					Style:  core.StyleLine,
				},
			},
		},
		{
			Overlay:   false,
			GroupName: "ATR",
			Time:      df.Time,
			Metrics: []core.IndicatorMetric{
				{
					Values: df.Metadata["atr"],
					Name:   "ATR",
					Color:  "blue",
					Style:  core.StyleLine,
				},
			},
		},
	}
}

// updateHigherTimeframeCache updates the higher timeframe dataframe cache
func (t *TrendMaster) updateHigherTimeframeCache(df *core.Dataframe) {
	// If we're not using higher timeframe confirmation, no need to update the cache
	if !t.config.UseHigherTfConfirmation {
		return
	}

	// Check if it's time to update the cache (every 5 minutes)
	now := time.Now()
	lastUpdate, exists := t.higherTfLastUpdate[df.Pair]
	if exists && now.Sub(lastUpdate) < time.Minute*5 {
		return
	}

	// Create new dataframe for higher timeframe if it doesn't exist
	if _, exists := t.higherTfCache[df.Pair]; !exists {
		t.higherTfCache[df.Pair] = &core.Dataframe{
			Pair:     df.Pair,
			Metadata: make(map[string]core.Series[float64]),
		}
	}

	// Process data for higher timeframe
	t.processHigherTimeframeData(df)

	// Update last update timestamp
	t.higherTfLastUpdate[df.Pair] = now
}

// processHigherTimeframeData processes data for the higher timeframe dataframe
func (t *TrendMaster) processHigherTimeframeData(df *core.Dataframe) {
	higherDf := t.higherTfCache[df.Pair]

	// Timeframe mapping to minutes
	timeframeMap := map[string]int{
		"1m":  1,
		"3m":  3,
		"5m":  5,
		"15m": 15,
		"30m": 30,
		"1h":  60,
		"2h":  120,
		"4h":  240,
		"6h":  360,
		"8h":  480,
		"12h": 720,
		"1d":  1440,
	}

	// Get number of minutes for each timeframe
	baseMinutes, existsBase := timeframeMap[t.config.Timeframe]
	higherMinutes, existsHigher := timeframeMap[t.config.HigherTimeframe]

	if !existsBase || !existsHigher {
		return // Invalid timeframes
	}

	// Number of base timeframe candles that form one higher timeframe candle
	candlesPerHigherTf := higherMinutes / baseMinutes

	// Ensure we have enough data
	if len(df.Close) < candlesPerHigherTf {
		return
	}

	// Group base timeframe candles into higher timeframe candles
	numCandles := len(df.Close) / candlesPerHigherTf

	// Initialize slices for the higher timeframe dataframe
	higherDf.Time = make([]time.Time, numCandles)
	higherDf.Open = make(core.Series[float64], numCandles)
	higherDf.High = make(core.Series[float64], numCandles)
	higherDf.Low = make(core.Series[float64], numCandles)
	higherDf.Close = make(core.Series[float64], numCandles)
	higherDf.Volume = make(core.Series[float64], numCandles)

	// Fill the higher timeframe dataframe
	for i := 0; i < numCandles; i++ {
		startIdx := i * candlesPerHigherTf
		endIdx := (i + 1) * candlesPerHigherTf
		if endIdx > len(df.Close) {
			endIdx = len(df.Close)
		}

		// Current period of candles
		periodOpen := df.Open[startIdx]
		periodClose := df.Close[endIdx-1]
		periodVolume := 0.0

		// Find highs and lows in the period
		periodHigh := df.High[startIdx]
		periodLow := df.Low[startIdx]

		for j := startIdx; j < endIdx; j++ {
			if df.High[j] > periodHigh {
				periodHigh = df.High[j]
			}
			if df.Low[j] < periodLow {
				periodLow = df.Low[j]
			}
			periodVolume += df.Volume[j]
		}

		// Fill the higher timeframe dataframe
		higherDf.Time[i] = df.Time[startIdx]
		higherDf.Open[i] = periodOpen
		higherDf.High[i] = periodHigh
		higherDf.Low[i] = periodLow
		higherDf.Close[i] = periodClose
		higherDf.Volume[i] = periodVolume
	}

	// Calculate indicators for the higher timeframe
	t.calculateHigherTimeframeIndicators(higherDf)
}

// calculateHigherTimeframeIndicators calculates indicators for the higher timeframe
func (t *TrendMaster) calculateHigherTimeframeIndicators(df *core.Dataframe) {
	df.Metadata["ema_fast"] = indicator.EMA(df.Close, t.config.EmaFastPeriod)
	df.Metadata["ema_slow"] = indicator.EMA(df.Close, t.config.EmaSlowPeriod)
	df.Metadata["ema_long"] = indicator.EMA(df.Close, t.config.EmaLongPeriod)
	df.Metadata["macd"], df.Metadata["macd_signal"], df.Metadata["macd_hist"] = indicator.MACD(
		df.Close,
		t.config.MacdFastPeriod,
		t.config.MacdSlowPeriod,
		t.config.MacdSignalPeriod,
	)
	df.Metadata["adx"] = indicator.ADX(df.High, df.Low, df.Close, t.config.AdxPeriod)
	df.Metadata["plus_di"] = talib.PlusDI(df.High, df.Low, df.Close, t.config.AdxPeriod)
	df.Metadata["minus_di"] = talib.MinusDI(df.High, df.Low, df.Close, t.config.AdxPeriod)
}

// updateCorrelationData updates the correlation data with the global market
func (t *TrendMaster) updateCorrelationData(df *core.Dataframe) {
	// If we're not using correlation, return
	if !t.config.UseMarketCorrelation {
		return
	}

	// Check if we have a valid close price
	if len(df.Close) == 0 {
		return
	}

	// Add last close price to the values array
	lastClose := df.Close.Last(0)
	if lastClose > 0 {
		t.correlationValues[df.Pair] = append(t.correlationValues[df.Pair], lastClose)

		// Limit the array size
		if len(t.correlationValues[df.Pair]) > t.config.CorrelationPeriod*2 {
			t.correlationValues[df.Pair] = t.correlationValues[df.Pair][1:]
		}
	}

	// If this pair contains the reference symbol, update the reference data
	if strings.Contains(df.Pair, t.config.CorrelationReferenceSymbol) {
		if lastClose > 0 {
			t.correlationRef[df.Pair] = append(t.correlationRef[df.Pair], lastClose)

			// Limit the array size
			if len(t.correlationRef[df.Pair]) > t.config.CorrelationPeriod*2 {
				t.correlationRef[df.Pair] = t.correlationRef[df.Pair][1:]
			}
		}

		// Update correlation for all pairs
		t.calculateAllCorrelations()
	}
}

// calculateAllCorrelations calculates the correlation between all pairs and the reference pair
func (t *TrendMaster) calculateAllCorrelations() {
	// Find the reference pair
	var refPair string
	var refValues []float64

	for pair, values := range t.correlationRef {
		if strings.Contains(pair, t.config.CorrelationReferenceSymbol) && len(values) >= t.config.CorrelationPeriod {
			refPair = pair
			refValues = values
			break
		}
	}

	// If we didn't find the reference pair with enough data, return
	if refPair == "" || len(refValues) < t.config.CorrelationPeriod {
		return
	}

	// Use only the last N values
	refValues = refValues[len(refValues)-t.config.CorrelationPeriod:]

	// Calculate correlation for each pair
	for pair, values := range t.correlationValues {
		// If we don't have enough data, skip
		if len(values) < t.config.CorrelationPeriod {
			continue
		}

		// Use only the last N values
		pairValues := values[len(values)-t.config.CorrelationPeriod:]

		// Calculate correlation
		correlation := t.calculateCorrelation(refValues, pairValues)

		// Store correlation
		t.marketCorrelation[pair] = correlation
	}
}

// calculateCorrelation calculates the correlation between two arrays of values
func (t *TrendMaster) calculateCorrelation(x, y []float64) float64 {
	// Check if the arrays have the same size
	if len(x) != len(y) {
		return 0
	}

	// Calculate means
	var sumX, sumY, sumXY, sumX2, sumY2 float64
	n := float64(len(x))

	for i := range x {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
		sumY2 += y[i] * y[i]
	}

	// Calculate Pearson correlation
	numerator := sumXY - sumX*sumY/n
	denominator := math.Sqrt((sumX2 - sumX*sumX/n) * (sumY2 - sumY*sumY/n))

	if denominator == 0 {
		return 0
	}

	return numerator / denominator
}

// OnCandle is called for each new candle and implements the trading logic
func (t *TrendMaster) OnCandle(ctx context.Context, df *core.Dataframe, broker core.Broker) {
	pair := df.Pair
	closePrice := df.Close.Last(0)

	// Initialize dataframe if needed
	t.initializeDataFrameIfNeeded(df)

	// Get current date (last candle)
	currentDate := t.getCurrentDate(df)

	// Reset daily trade count if we're in a new day
	t.resetDailyTradeCountIfNewDay(currentDate)

	// Check if we're within allowed trading hours
	if !t.isWithinTradingHours() {
		return
	}

	// Get current position
	assetPosition, quotePosition, err := broker.Position(ctx, pair)
	if err != nil {
		bot.DefaultLog.Error(err)
		return
	}

	// Detect market type
	t.detectMarketType(pair)

	// Update trailing stop if in position
	t.updateTrailingStopIfInPosition(pair, closePrice)

	// Check entry and exit signals
	if assetPosition > 0 {
		// We're already in a long position, check partial exit
		if t.config.UsePartialTakeProfit {
			t.checkPartialExits(ctx, df, broker, assetPosition, pair)
		}

		// Check complete exit
		if t.shouldExit(df) || t.checkTrailingStop(df, pair) {
			t.executeExit(ctx, df, broker, assetPosition)
			// Reset after exit
			delete(t.lastPrice, pair)
		}
	} else {
		// No position, check entry (only if we haven't exceeded the daily limit)
		tradeCount := t.dailyTradeCount[pair]
		if tradeCount < t.config.MaxTradesPerDay && t.shouldEnter(df) {
			t.executeEntry(ctx, df, broker, quotePosition, closePrice)
			// Increment trade counter
			t.dailyTradeCount[pair] = tradeCount + 1
		}
	}

	// Update date cache (for the next day check)
	t.lastTradeDate = currentDate
}

// initializeDataFrameIfNeeded initializes dataframe-specific data if needed
func (t *TrendMaster) initializeDataFrameIfNeeded(df *core.Dataframe) {
	pair := df.Pair
	if !t.isDataFrameInitialized[pair] {
		t.isDataFrameInitialized[pair] = true

		// Initialize arrays for correlation
		t.correlationValues[pair] = make([]float64, 0, t.config.CorrelationPeriod*2)

		// If we're using correlation and this is the reference pair
		if t.config.UseMarketCorrelation && strings.Contains(pair, t.config.CorrelationReferenceSymbol) {
			t.correlationRef[pair] = make([]float64, 0, t.config.CorrelationPeriod*2)
		}
	}
}

// getCurrentDate gets the current date from the dataframe
func (t *TrendMaster) getCurrentDate(df *core.Dataframe) string {
	if len(df.Time) > 0 {
		return df.Time[len(df.Time)-1].Format("2006-01-02")
	}
	return ""
}

// resetDailyTradeCountIfNewDay resets the daily trade count if we're in a new day
func (t *TrendMaster) resetDailyTradeCountIfNewDay(currentDate string) {
	if currentDate != t.lastTradeDate {
		t.dailyTradeCount = make(map[string]int)
		t.lastTradeDate = currentDate
	}
}

// isWithinTradingHours checks if we're within allowed trading hours
func (t *TrendMaster) isWithinTradingHours() bool {
	if !t.config.TradingHoursEnabled {
		return true
	}

	currentHour := time.Now().UTC().Hour()
	return currentHour >= t.config.TradingStartHour && currentHour < t.config.TradingEndHour
}

// updateTrailingStopIfInPosition updates the trailing stop if we're in a position
func (t *TrendMaster) updateTrailingStopIfInPosition(pair string, closePrice float64) {
	lastPrice, exists := t.lastPrice[pair]
	if exists && closePrice > lastPrice {
		t.lastPrice[pair] = closePrice
	}
}

// detectMarketType detects the market type based on the pair
func (t *TrendMaster) detectMarketType(pair string) {
	// If we already have the market type stored, use the stored value
	if _, exists := t.marketType[pair]; exists {
		return
	}

	// Detect market type based on the pair pattern
	if strings.HasSuffix(pair, "USDT") || strings.HasSuffix(pair, "BTC") || strings.HasSuffix(pair, "ETH") ||
		strings.HasSuffix(pair, "BNB") || strings.Contains(pair, "PERP") {
		t.marketType[pair] = "crypto"
	} else if strings.HasSuffix(pair, "USD") || strings.Contains(pair, "JPY") ||
		strings.Contains(pair, "EUR") || strings.Contains(pair, "GBP") ||
		strings.Contains(pair, "AUD") || strings.Contains(pair, "CAD") ||
		strings.Contains(pair, "CHF") || strings.Contains(pair, "NZD") {
		t.marketType[pair] = "forex"
	} else {
		t.marketType[pair] = "stocks"
	}
}

// getMarketSpecificConfig returns the configuration specific to the market type
func (t *TrendMaster) getMarketSpecificConfig(pair string) MarketSpecificConfig {
	marketType, exists := t.marketType[pair]
	if !exists {
		// Default market type
		marketType = "crypto"
	}

	// Check if we have a specific configuration for this market type
	if config, exists := t.config.MarketSpecificSettings[marketType]; exists {
		return config
	}

	// Return default configuration
	return MarketSpecificConfig{
		VolatilityThreshold: t.config.VolatilityThreshold,
		TrailingStopPercent: t.config.TrailingStopPercent,
		AtrMultiplier:       t.config.AtrMultiplier,
	}
}

// checkTrailingStop checks if the trailing stop was hit
func (t *TrendMaster) checkTrailingStop(df *core.Dataframe, pair string) bool {
	closePrice := df.Close.Last(0)
	lastPrice, exists := t.lastPrice[pair]

	if !exists {
		return false
	}

	// Get market-specific configuration
	marketConfig := t.getMarketSpecificConfig(pair)
	trailStopPercent := marketConfig.TrailingStopPercent

	// If the price dropped below the trailing stop, trigger exit
	trailAmount := lastPrice * trailStopPercent
	if closePrice <= lastPrice-trailAmount {
		bot.DefaultLog.WithFields(map[string]any{
			"pair":         pair,
			"highestPrice": lastPrice,
			"currentPrice": closePrice,
			"trailAmount":  trailAmount,
		}).Info("Trailing stop activated")
		return true
	}

	return false
}

// checkHigherTimeframeTrend checks the trend in the higher timeframe
func (t *TrendMaster) checkHigherTimeframeTrend(pair string) bool {
	// If we're not using higher timeframe filter, return true
	if !t.config.UseHigherTfConfirmation {
		return true
	}

	// Check if we have higher timeframe cache
	higherDf, exists := t.higherTfCache[pair]
	if !exists || len(higherDf.Close) == 0 {
		return true // Not enough data, allow entry
	}

	// Check conditions in the higher timeframe
	emaFast := higherDf.Metadata["ema_fast"].Last(0)
	emaSlow := higherDf.Metadata["ema_slow"].Last(0)
	emaLong := higherDf.Metadata["ema_long"].Last(0)
	macd := higherDf.Metadata["macd"].Last(0)
	macdSignal := higherDf.Metadata["macd_signal"].Last(0)
	plusDI := higherDf.Metadata["plus_di"].Last(0)
	minusDI := higherDf.Metadata["minus_di"].Last(0)
	adx := higherDf.Metadata["adx"].Last(0)

	// Check for uptrend in higher timeframe
	emaAlignment := emaFast > emaSlow && emaSlow > emaLong
	macdBullish := macd > macdSignal
	diPositive := plusDI > minusDI
	strongTrend := adx > t.config.AdxThreshold

	// At least 3 of 4 conditions must be true
	conditionsCount := 0
	if emaAlignment {
		conditionsCount++
	}
	if macdBullish {
		conditionsCount++
	}
	if diPositive {
		conditionsCount++
	}
	if strongTrend {
		conditionsCount++
	}

	return conditionsCount >= 3
}

// shouldEnter checks if entry conditions are met
func (t *TrendMaster) shouldEnter(df *core.Dataframe) bool {
	pair := df.Pair
	closePrice := df.Close.Last(0)
	emaLongHigh := df.Metadata["ema_long_high"].Last(0)
	emaFast := df.Metadata["ema_fast"].Last(0)
	emaSlow := df.Metadata["ema_slow"].Last(0)
	macd := df.Metadata["macd"].Last(0)
	macdSignal := df.Metadata["macd_signal"].Last(0)
	plusDI := df.Metadata["plus_di"].Last(0)
	minusDI := df.Metadata["minus_di"].Last(0)
	adx := df.Metadata["adx"].Last(0)
	atr := df.Metadata["atr"].Last(0)

	// Basic checks (original conditions)
	priceAboveEMA := closePrice > emaLongHigh
	macdAboveSignal := macd > macdSignal
	plusDIAboveMinusDI := plusDI > minusDI
	adxAboveThreshold := adx > t.config.AdxThreshold
	emaFastAboveSlow := emaFast > emaSlow
	diSpreadSufficient := (plusDI - minusDI) > t.config.AdxMinimumDiSpread

	// Get market-specific configuration
	marketConfig := t.getMarketSpecificConfig(pair)

	// Volatility filter: check if volatility is not too high
	volatilityCheck := true
	if atr > 0 {
		volatilityRatio := atr / closePrice
		volatilityCheck = volatilityRatio < marketConfig.VolatilityThreshold
	}

	// RSI filter: avoid buying in overbought market
	rsiCheck := true
	if t.config.UseRsiFilter {
		rsi := df.Metadata["rsi"].Last(0)
		rsiCheck = rsi < t.config.RsiOverbought // RSI not overbought
	}

	// Volume filter: check if volume is sufficient
	volumeCheck := true
	if t.config.UseVolFilter {
		volume := df.Volume.Last(0)
		volumeAvg := df.Metadata["vol_avg"].Last(0)
		if volumeAvg > 0 {
			volumeRatio := volume / volumeAvg
			volumeCheck = volumeRatio >= t.config.VolMinRatio // Volume above average
		}
	}

	// Check higher timeframe confirmation
	higherTfCheck := t.checkHigherTimeframeTrend(pair)

	// Check market sentiment filter
	sentimentCheck := true
	if t.config.UseSentimentFilter {
		sentiment, exists := t.marketSentiment[pair]
		if exists {
			// Lower values are more restrictive (more fear in the market)
			if sentiment < t.config.SentimentThreshold {
				// In high fear markets, require stronger conditions
				sentimentCheck = adx > (t.config.AdxThreshold+5) && (plusDI-minusDI) > 15
			}
		}
	}

	// Check market correlation
	correlationCheck := true
	if t.config.UseMarketCorrelation {
		correlation, exists := t.marketCorrelation[pair]
		if exists {
			// If strongly negative correlation with the market, be more cautious
			if correlation < t.config.NegativeCorrelationThreshold {
				correlationCheck = false // Avoid trading against the global market
			}
		}
	}

	// Check for consistently bullish market
	bullishMarket := true

	// Reduce trading frequency after consecutive losses
	riskAdjustment := true
	if t.consecutiveLosses >= 2 {
		// After 2 consecutive losses, require stricter conditions
		riskAdjustment = adx > (t.config.AdxThreshold+5) && (plusDI-minusDI) > 10
	}

	// All main conditions must be true for entry
	mainConditions := priceAboveEMA && macdAboveSignal && plusDIAboveMinusDI &&
		adxAboveThreshold && emaFastAboveSlow && diSpreadSufficient

	// Additional filters to improve signal quality
	additionalFilters := volatilityCheck && rsiCheck && volumeCheck &&
		bullishMarket && riskAdjustment && higherTfCheck &&
		sentimentCheck && correlationCheck

	return mainConditions && additionalFilters
}

// shouldExit checks if exit conditions are met
func (t *TrendMaster) shouldExit(df *core.Dataframe) bool {
	closePrice := df.Close.Last(0)
	emaLongLow := df.Metadata["ema_long_low"].Last(0)
	emaFast := df.Metadata["ema_fast"].Last(0)
	emaSlow := df.Metadata["ema_slow"].Last(0)
	macd := df.Metadata["macd"].Last(0)
	macdSignal := df.Metadata["macd_signal"].Last(0)
	plusDI := df.Metadata["plus_di"].Last(0)
	minusDI := df.Metadata["minus_di"].Last(0)
	adx := df.Metadata["adx"].Last(0)

	// Basic exit conditions
	priceBelowEMA := closePrice < emaLongLow
	macdBelowSignal := macd < macdSignal
	minusDIAbovePlusDI := minusDI > plusDI
	adxAboveThreshold := adx > t.config.AdxThreshold
	emaFastBelowSlow := emaFast < emaSlow

	// Check for break of long EMA calculated on lows
	// This is a quick exit signal independent of other conditions
	if priceBelowEMA && t.config.UsePriceActionExit {
		bot.DefaultLog.WithFields(map[string]any{
			"pair":       df.Pair,
			"closePrice": closePrice,
			"emaLongLow": emaLongLow,
		}).Info("Quick exit: price below EMA Low")
		return true
	}

	// Quick exit if MACD falls very rapidly (potential strong reversal)
	if t.config.UseMacdReversalExit {
		macdHist := df.Metadata["macd_hist"].Last(0)
		prevMacdHist := df.Metadata["macd_hist"].Last(1)
		if macdHist < 0 && prevMacdHist > 0 && macdHist < prevMacdHist*-t.config.MacdReversalThreshold {
			bot.DefaultLog.WithFields(map[string]any{
				"pair":         df.Pair,
				"macdHist":     macdHist,
				"prevMacdHist": prevMacdHist,
			}).Info("Quick exit: MACD strongly reversing")
			return true
		}
	}

	// Quick exit if ADX starts falling while in position (weakening trend)
	if t.config.UseAdxFallingExit {
		adxFalling := adx < df.Metadata["adx"].Last(1) && df.Metadata["adx"].Last(1) < df.Metadata["adx"].Last(2)
		if adxFalling && minusDIAbovePlusDI {
			bot.DefaultLog.WithFields(map[string]any{
				"pair":    df.Pair,
				"adx":     adx,
				"prevAdx": df.Metadata["adx"].Last(1),
				"plusDI":  plusDI,
				"minusDI": minusDI,
			}).Info("Quick exit: ADX falling and -DI above +DI")
			return true
		}
	}

	// RSI indicating extreme overbought
	if t.config.UseRsiFilter {
		rsi := df.Metadata["rsi"].Last(0)
		if rsi > t.config.RsiExtremeOverbought { // Extreme overbought
			bot.DefaultLog.WithFields(map[string]any{
				"pair": df.Pair,
				"rsi":  rsi,
			}).Info("Quick exit: RSI in extreme overbought")
			return true
		}
	}

	// All regular exit conditions
	regularExitConditions := macdBelowSignal && minusDIAbovePlusDI && adxAboveThreshold && emaFastBelowSlow

	return regularExitConditions
}

// checkPartialExits checks and executes partial exits
func (t *TrendMaster) checkPartialExits(ctx context.Context, df *core.Dataframe, broker core.Broker, assetPosition float64, pair string) {
	// If we're not using partial exits or don't have configured levels, return
	if !t.config.UsePartialTakeProfit || len(t.config.PartialExitLevels) == 0 {
		return
	}

	closePrice := df.Close.Last(0)
	entryPrice, exists := t.entryPrice[pair]

	// If we don't have an entry price, we can't check partial exits
	if !exists {
		return
	}

	// Check if we already have partial positions for this pair
	positions, posExists := t.partialPositions[pair]
	if !posExists {
		// Create array to track partial positions
		t.partialPositions[pair] = make([]PartialPosition, 0)

		// For each configured level, create a partial position
		totalPosition := assetPosition
		for i, level := range t.config.PartialExitLevels {
			// Calculate quantity for this level
			levelQuantity := assetPosition * level.Percentage

			// Create partial position
			partialPos := PartialPosition{
				Quantity:   levelQuantity,
				EntryPrice: entryPrice,
				Level:      i,
			}

			// Add to the list of partial positions
			t.partialPositions[pair] = append(t.partialPositions[pair], partialPos)

			// Check if we've already reached the target for this level
			if !level.TrailingOnly {
				targetPrice := entryPrice * (1.0 + level.Target)

				// If the current price has already reached the target, execute partial exit
				if closePrice >= targetPrice {
					t.executePartialExit(ctx, df, broker, pair, i, partialPos.Quantity, "Target reached")
				}
			}

			totalPosition -= levelQuantity
		}

		positions = t.partialPositions[pair]
	}

	// For each existing partial position
	for i, pos := range positions {
		// Skip positions already executed
		if pos.Quantity <= 0 {
			continue
		}

		// Get configured level
		level := t.config.PartialExitLevels[pos.Level]

		// If not trailing only, check price target
		if !level.TrailingOnly {
			targetPrice := pos.EntryPrice * (1.0 + level.Target)

			// If the current price has reached the target, execute partial exit
			if closePrice >= targetPrice {
				t.executePartialExit(ctx, df, broker, pair, i, pos.Quantity, "Target reached")
			}
		} else {
			// For trailing-only levels, check trailing stop
			// Use the last highest price recorded
			lastHighestPrice, exists := t.lastPrice[pair]
			if exists {
				// Calculate trailing stop price
				trailAmount := lastHighestPrice * t.config.TrailingStopPercent
				trailingStopPrice := lastHighestPrice - trailAmount

				// If the current price has fallen below the trailing stop, execute partial exit
				if closePrice <= trailingStopPrice {
					t.executePartialExit(ctx, df, broker, pair, i, pos.Quantity, "Trailing stop hit")
				}
			}
		}
	}
}

// executePartialExit executes a partial exit
func (t *TrendMaster) executePartialExit(ctx context.Context, df *core.Dataframe, broker core.Broker, pair string, posIndex int, quantity float64, reason string) {
	// Don't execute if quantity is zero or negative
	if quantity <= 0 {
		return
	}

	// Execute market sell order for the partial quantity
	order, err := broker.CreateOrderMarket(ctx, core.SideTypeSell, pair, quantity)
	if err != nil {
		bot.DefaultLog.WithFields(map[string]any{
			"pair":     pair,
			"quantity": quantity,
			"level":    posIndex,
			"error":    err,
		}).Error("Failed to execute partial exit")
		return
	}

	// Update partial position
	t.partialPositions[pair][posIndex].Quantity = 0
	t.partialPositions[pair][posIndex].OrderID = order.ID

	// Log partial exit
	bot.DefaultLog.WithFields(map[string]any{
		"pair":     pair,
		"quantity": quantity,
		"level":    posIndex,
		"orderID":  order.ID,
		"reason":   reason,
		"price":    df.Close.Last(0),
	}).Info("Partial exit executed")
}

// executeEntry executes the entry operation
func (t *TrendMaster) executeEntry(ctx context.Context, df *core.Dataframe, broker core.Broker, quotePosition, closePrice float64) {
	pair := df.Pair

	// Cancel any previous active orders
	if ordersMap, exists := t.activeOrders[pair]; exists {
		for _, orderID := range ordersMap {
			order, err := broker.Order(ctx, pair, orderID)
			if err == nil {
				_ = broker.Cancel(ctx, order)
			}
		}
		delete(t.activeOrders, pair)
	}

	// Get market-specific configuration
	marketConfig := t.getMarketSpecificConfig(pair)

	// Adjust position size based on ATR for risk control
	atr := df.Metadata["atr"].Last(0)
	positionSize := t.config.PositionSize

	// Adjust position size based on recent performance (if enabled)
	if t.config.UseAdaptiveSize {
		// Increase size after consecutive wins
		if t.winStreak > 0 {
			increaseFactor := math.Min(float64(t.winStreak)*t.config.WinIncreaseFactor,
				t.config.MaxPositionSizeFactor-1.0)
			positionSize *= (1.0 + increaseFactor)

			// Limit to maximum size
			if positionSize > t.config.PositionSize*t.config.MaxPositionSizeFactor {
				positionSize = t.config.PositionSize * t.config.MaxPositionSizeFactor
			}
		}

		// Reduce size after consecutive losses
		if t.consecutiveLosses > 0 {
			// Reduce size for each consecutive loss
			reductionFactor := math.Max(1.0-float64(t.consecutiveLosses)*t.config.LossReductionFactor, t.config.MinPositionSizeFactor)
			positionSize *= reductionFactor
		}
	}

	// Calculate stop loss based on ATR if available
	stopLossPrice := 0.0
	if atr > 0 {
		// Use ATR to calculate dynamic stop loss
		stopLossPrice = closePrice - (atr * marketConfig.AtrMultiplier)

		// Calculate stop loss percentage based on ATR
		stopLossPercent := (closePrice - stopLossPrice) / closePrice

		// If ATR-based stop loss is greater than maximum allowed, adjust position size
		if stopLossPercent > t.config.MaxRiskPerTrade {
			// Adjust position size to limit risk
			riskAdjustment := t.config.MaxRiskPerTrade / stopLossPercent
			positionSize *= riskAdjustment
		}
	} else {
		// Use fixed stop loss if ATR is not available
		stopLossPrice = closePrice * (1.0 - t.config.MaxRiskPerTrade)
	}

	// Limit risk per trade
	maxRiskAmount := quotePosition * t.config.MaxRiskPerTrade
	actualRiskAmount := quotePosition * positionSize * ((closePrice - stopLossPrice) / closePrice)
	if actualRiskAmount > maxRiskAmount {
		// Adjust position size to limit risk
		positionSize *= maxRiskAmount / actualRiskAmount
	}

	// Calculate position size based on available capital
	entryAmount := quotePosition * positionSize

	// Execute market buy order
	_, err := broker.CreateOrderMarketQuote(ctx, core.SideTypeBuy, pair, entryAmount)
	if err != nil {
		bot.DefaultLog.WithFields(map[string]any{
			"pair":  pair,
			"side":  core.SideTypeBuy,
			"quote": entryAmount,
			"price": closePrice,
		}).Error(err)
		return
	}

	// Record entry price
	t.entryPrice[pair] = closePrice

	// Initialize trailing stop with entry price
	t.lastPrice[pair] = closePrice

	// Store position size for use in partial exits
	t.positionSize[pair] = positionSize

	// Get updated position after purchase
	assetPosition, _, err := broker.Position(ctx, pair)
	if err != nil {
		bot.DefaultLog.Error(err)
		return
	}

	// Create map for orders for this pair if it doesn't exist
	if _, exists := t.activeOrders[pair]; !exists {
		t.activeOrders[pair] = make(map[int]int64)
	}

	// If partial exit is enabled, initialize structures
	if t.config.UsePartialTakeProfit {
		// Clear any old data
		t.partialPositions[pair] = make([]PartialPosition, 0)
		t.partialOrders[pair] = make([]int64, 0)

		// Partial positions will be configured in the next execution of checkPartialExits
	} else {
		// Set take profit and stop loss with OCO order
		takeProfitPrice := closePrice * (1.0 + t.calculateTakeProfit(df, pair))

		orders, err := broker.CreateOrderOCO(
			ctx,
			core.SideTypeSell,
			pair,
			assetPosition,
			takeProfitPrice,
			stopLossPrice,
			stopLossPrice,
		)

		if err != nil {
			bot.DefaultLog.WithFields(map[string]any{
				"pair":       pair,
				"side":       core.SideTypeSell,
				"asset":      assetPosition,
				"takeProfit": takeProfitPrice,
				"stopPrice":  stopLossPrice,
			}).Error(err)
		} else if len(orders) > 0 {
			// Store order IDs for future reference
			for i, order := range orders {
				t.activeOrders[pair][i] = order.ID
			}
		}
	}

	bot.DefaultLog.WithFields(map[string]any{
		"pair":              pair,
		"entryPrice":        closePrice,
		"positionSize":      positionSize,
		"stopLossPrice":     stopLossPrice,
		"takeProfitPrice":   closePrice * (1.0 + t.calculateTakeProfit(df, pair)),
		"consecutiveLosses": t.consecutiveLosses,
	}).Info("Entry executed")
}

// calculateTakeProfit calculates the profit target based on settings
func (t *TrendMaster) calculateTakeProfit(df *core.Dataframe, pair string) float64 {
	// If dynamic targets are not enabled, use base target
	if !t.config.UseDynamicTargets {
		return t.config.BaseTarget
	}

	// Calculate target based on ATR
	atr := df.Metadata["atr"].Last(0)
	closePrice := df.Close.Last(0)

	if atr <= 0 || closePrice <= 0 {
		return t.config.BaseTarget
	}

	// Calculate dynamic target based on volatility (ATR)
	atrPercent := atr / closePrice
	dynamicTarget := atrPercent * t.config.AtrTargetFactor

	// Limit between configured minimum and maximum
	if dynamicTarget < t.config.MinTarget {
		dynamicTarget = t.config.MinTarget
	} else if dynamicTarget > t.config.MaxTarget {
		dynamicTarget = t.config.MaxTarget
	}

	return dynamicTarget
}

// executeExit executes the exit operation
func (t *TrendMaster) executeExit(ctx context.Context, df *core.Dataframe, broker core.Broker, assetPosition float64) {
	pair := df.Pair
	currentPrice := df.Close.Last(0)

	// Cancel all active orders
	if ordersMap, exists := t.activeOrders[pair]; exists {
		for _, orderID := range ordersMap {
			order, err := broker.Order(ctx, pair, orderID)
			if err == nil {
				err = broker.Cancel(ctx, order)
				if err != nil {
					bot.DefaultLog.WithFields(map[string]any{
						"pair":    pair,
						"orderID": orderID,
					}).Error(err)
				}
			}
		}
		delete(t.activeOrders, pair)
	}

	// Cancel partial exit orders
	if orders, exists := t.partialOrders[pair]; exists && len(orders) > 0 {
		for _, orderID := range orders {
			order, err := broker.Order(ctx, pair, orderID)
			if err == nil {
				_ = broker.Cancel(ctx, order)
			}
		}
		delete(t.partialOrders, pair)
	}

	// Sell entire position
	_, err := broker.CreateOrderMarket(ctx, core.SideTypeSell, pair, assetPosition)
	if err != nil {
		bot.DefaultLog.WithFields(map[string]any{
			"pair":  pair,
			"side":  core.SideTypeSell,
			"asset": assetPosition,
			"price": currentPrice,
		}).Error(err)
		return
	}

	// Check trade result
	entryPrice, exists := t.entryPrice[pair]
	if exists {
		tradeProfit := currentPrice > entryPrice
		t.lastTradeResult[pair] = tradeProfit

		// Update win/loss counters
		if tradeProfit {
			t.winCount++
			t.winStreak++
			t.lossStreak = 0
			t.consecutiveLosses = 0 // Reset after profitable trade
		} else {
			t.lossCount++
			t.lossStreak++
			t.winStreak = 0
			t.consecutiveLosses++ // Increment consecutive losses counter
		}

		profitPercent := (currentPrice - entryPrice) / entryPrice * 100

		bot.DefaultLog.WithFields(map[string]any{
			"pair":              pair,
			"entryPrice":        entryPrice,
			"exitPrice":         currentPrice,
			"profit":            profitPercent,
			"isProfit":          tradeProfit,
			"consecutiveLosses": t.consecutiveLosses,
			"winRate":           float64(t.winCount) / float64(t.winCount+t.lossCount) * 100,
		}).Info("Exit executed")

		// Clear entry price
		delete(t.entryPrice, pair)
	}

	// Clear other tracking data for this pair
	delete(t.positionSize, pair)
	delete(t.partialPositions, pair)
}
