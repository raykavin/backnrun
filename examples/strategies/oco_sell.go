package strategies

import (
	"github.com/markcheno/go-talib"
	"github.com/raykavin/backnrun"
	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/indicator"
)

// OCOSell implements a trading strategy using Stochastic oscillator
// for entry signals and OCO (One-Cancels-Other) orders for exits
type OCOSell struct {
	// Configuration parameters (can be made configurable)
	buyAmount    float64
	stochPeriod  int
	stochKPeriod int
	stochDPeriod int
	profitTarget float64
	stopLoss     float64
}

// NewOCOSell creates a new instance of the OCOSell strategy with default parameters
func NewOCOSell() *OCOSell {
	return &OCOSell{
		buyAmount:    4000.0,
		stochPeriod:  8,
		stochKPeriod: 3,
		stochDPeriod: 3,
		profitTarget: 1.1,  // 10% profit target
		stopLoss:     0.95, // 5% stop loss
	}
}

// Timeframe returns the required timeframe for this strategy
func (s OCOSell) Timeframe() string {
	return "5m"
}

// WarmupPeriod returns the number of candles needed before the strategy is ready
func (s OCOSell) WarmupPeriod() int {
	return 9 // Matches the stochastic parameters
}

// Indicators calculates and returns the indicators used by this strategy
func (s OCOSell) Indicators(df *core.Dataframe) []core.ChartIndicator {
	// Calculate Stochastic oscillator
	df.Metadata["stoch"], df.Metadata["stoch_signal"] = indicator.Stoch(
		df.High,
		df.Low,
		df.Close,
		s.stochPeriod,
		s.stochKPeriod,
		talib.SMA,
		s.stochDPeriod,
		talib.SMA,
	)

	// Return chart indicators for visualization
	return []core.ChartIndicator{
		{
			Overlay:   false,
			GroupName: "Stochastic",
			Time:      df.Time,
			Metrics: []core.IndicatorMetric{
				{
					Values: df.Metadata["stoch"],
					Name:   "K",
					Color:  "red",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["stoch_signal"],
					Name:   "D",
					Color:  "blue",
					Style:  core.StyleLine,
				},
			},
		},
	}
}

// OnCandle is called for each new candle and implements the trading logic
func (s *OCOSell) OnCandle(df *core.Dataframe, broker core.Broker) {
	closePrice := df.Close.Last(0)
	backnrun.DefaultLog.Info("New Candle = ", df.Pair, df.LastUpdate, closePrice)

	// Get current position
	assetPosition, quotePosition, err := broker.Position(df.Pair)
	if err != nil {
		backnrun.DefaultLog.Error(err)
		return
	}

	// Check entry conditions: enough quote currency and stochastic K line crosses above D line
	if quotePosition > s.buyAmount && df.Metadata["stoch"].Crossover(df.Metadata["stoch_signal"]) {
		s.executeEntryAndExitStrategy(df, broker, closePrice, assetPosition, quotePosition)
	}
}

// executeEntryAndExitStrategy handles the order execution logic
func (s *OCOSell) executeEntryAndExitStrategy(
	df *core.Dataframe,
	broker core.Broker,
	closePrice float64,
	assetPosition float64,
	quotePosition float64,
) {
	// Calculate position size
	size := s.buyAmount / closePrice

	// Execute market buy order
	_, err := broker.CreateOrderMarket(core.SideTypeBuy, df.Pair, size)
	if err != nil {
		s.logOrderError(err, df.Pair, core.SideTypeBuy, closePrice, assetPosition, quotePosition, size)
		return
	}

	// Set take profit and stop loss with OCO order
	takeProfitPrice := closePrice * s.profitTarget
	stopLossPrice := closePrice * s.stopLoss

	_, err = broker.CreateOrderOCO(
		core.SideTypeSell,
		df.Pair,
		size,
		takeProfitPrice,
		stopLossPrice,
		stopLossPrice,
	)
	if err != nil {
		s.logOrderError(err, df.Pair, core.SideTypeSell, closePrice, assetPosition, quotePosition, size)
	}
}

// logOrderError provides consistent error logging for order failures
func (s *OCOSell) logOrderError(
	err error,
	pair string,
	side core.SideType,
	closePrice float64,
	assetPosition float64,
	quotePosition float64,
	size float64,
) {
	backnrun.DefaultLog.WithFields(map[string]interface{}{
		"pair":  pair,
		"side":  side,
		"close": closePrice,
		"asset": assetPosition,
		"quote": quotePosition,
		"size":  size,
	}).Error(err)
}
