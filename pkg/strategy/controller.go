package strategy

import (
	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/logger"
)

// Controller manages the execution of trading strategies
type Controller struct {
	strategy         core.Strategy
	dataframeManager *DataframeManager
	broker           core.Broker
	log              logger.Logger
	started          bool
}

// NewStrategyController creates a new strategy controller
func NewStrategyController(pair string, strategy core.Strategy, broker core.Broker, log logger.Logger) *Controller {
	return &Controller{
		dataframeManager: NewDataframeManager(pair),
		strategy:         strategy,
		broker:           broker,
		log:              log,
	}
}

// Start begins the strategy execution
func (c *Controller) Start() {
	c.started = true
}

// OnPartialCandle processes partial candle updates for high-frequency strategies
func (c *Controller) OnPartialCandle(candle core.Candle) {
	if !candle.Complete && c.dataframeManager.HasSufficientData(c.strategy.WarmupPeriod()) {
		if highFreqStrategy, ok := c.strategy.(core.HighFrequencyStrategy); ok {
			c.dataframeManager.UpdateDataFrame(candle)

			dataframe := c.dataframeManager.GetDataframe()
			highFreqStrategy.Indicators(dataframe)
			highFreqStrategy.OnPartialCandle(dataframe, c.broker)
		}
	}
}

// OnCandle processes completed candles for all strategy types
func (c *Controller) OnCandle(candle core.Candle) {
	if c.dataframeManager.IsLateCandle(candle) {
		c.log.Errorf("late candle received: %#v", candle)
		return
	}

	c.dataframeManager.UpdateDataFrame(candle)

	if c.dataframeManager.HasSufficientData(c.strategy.WarmupPeriod()) {
		sample := c.dataframeManager.GetSample(c.strategy.WarmupPeriod())
		c.strategy.Indicators(&sample)

		if c.started {
			c.strategy.OnCandle(&sample, c.broker)
		}
	}
}
