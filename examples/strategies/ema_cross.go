package strategies

import (
	"github.com/raykavin/backnrun"
	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/indicator"
	"github.com/raykavin/backnrun/pkg/strategy"
)

// CrossEMA implements a trading strategy using EMA and SMA crossovers
// for entry and exit signals
type CrossEMA struct {
	// Configuration parameters
	emaLength      int
	smaLength      int
	minQuoteAmount float64
}

// NewCrossEMA creates a new instance of the CrossEMA strategy with default parameters
func NewCrossEMA() *CrossEMA {
	return &CrossEMA{
		emaLength:      9,
		smaLength:      21,
		minQuoteAmount: 10.0,
	}
}

// Timeframe returns the required timeframe for this strategy
func (s CrossEMA) Timeframe() string {
	return "1m"
}

// WarmupPeriod returns the number of candles needed before the strategy is ready
func (s CrossEMA) WarmupPeriod() int {
	return 200
}

// Indicators calculates and returns the indicators used by this strategy
func (s CrossEMA) Indicators(df *core.Dataframe) []strategy.ChartIndicator {
	// Calculate indicators
	df.Metadata["ema9"] = indicator.EMA(df.Close, s.emaLength)
	df.Metadata["sma21"] = indicator.SMA(df.Close, s.smaLength)

	// Return chart indicators for visualization
	return []strategy.ChartIndicator{
		{
			Overlay:   true,
			GroupName: "MA's",
			Time:      df.Time,
			Metrics: []strategy.IndicatorMetric{
				{
					Values: df.Metadata["ema9"],
					Name:   "EMA " + string(rune(s.emaLength+'0')),
					Color:  "red",
					Style:  strategy.StyleLine,
				},
				{
					Values: df.Metadata["sma21"],
					Name:   "SMA " + string(rune(s.smaLength+'0')),
					Color:  "blue",
					Style:  strategy.StyleLine,
				},
			},
		},
	}
}

// OnCandle is called for each new candle and implements the trading logic
func (s *CrossEMA) OnCandle(df *core.Dataframe, broker core.Broker) {
	closePrice := df.Close.Last(0)

	// Get current position
	assetPosition, quotePosition, err := broker.Position(df.Pair)
	if err != nil {
		backnrun.DefaultLog.Error(err)
		return
	}

	// Check for buy signal: EMA9 crosses above SMA21
	if s.shouldBuy(df, quotePosition) {
		s.executeBuy(df, broker, closePrice, quotePosition)
		return
	}

	// Check for sell signal: EMA9 crosses below SMA21
	if s.shouldSell(df, assetPosition) {
		s.executeSell(df, broker, assetPosition)
	}
}

// shouldBuy checks if buying conditions are met
func (s *CrossEMA) shouldBuy(df *core.Dataframe, quotePosition float64) bool {
	return quotePosition >= s.minQuoteAmount &&
		df.Metadata["ema9"].Crossover(df.Metadata["sma21"])
}

// shouldSell checks if selling conditions are met
func (s *CrossEMA) shouldSell(df *core.Dataframe, assetPosition float64) bool {
	return assetPosition > 0 &&
		df.Metadata["ema9"].Crossunder(df.Metadata["sma21"])
}

// executeBuy performs the buy operation
func (s *CrossEMA) executeBuy(df *core.Dataframe, broker core.Broker, closePrice, quotePosition float64) {
	amount := quotePosition / closePrice // calculate amount of asset to buy
	_, err := broker.CreateOrderMarket(core.SideTypeBuy, df.Pair, amount)
	if err != nil {
		backnrun.DefaultLog.WithFields(map[string]interface{}{
			"pair":  df.Pair,
			"side":  core.SideTypeBuy,
			"price": closePrice,
			"size":  amount,
		}).Error(err)
	}
}

// executeSell performs the sell operation
func (s *CrossEMA) executeSell(df *core.Dataframe, broker core.Broker, assetPosition float64) {
	_, err := broker.CreateOrderMarket(core.SideTypeSell, df.Pair, assetPosition)
	if err != nil {
		backnrun.DefaultLog.WithFields(map[string]interface{}{
			"pair": df.Pair,
			"side": core.SideTypeSell,
			"size": assetPosition,
		}).Error(err)
	}
}
