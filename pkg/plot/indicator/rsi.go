package indicator

import (
	"fmt"

	"github.com/markcheno/go-talib"
	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/plot"
)

// RSI creates a new Relative Strength Index indicator
// period: the number of periods to use for calculations
// color: the color to use for the indicator line
func RSI(period int, color string) plot.Indicator {
	return &rsi{
		BaseIndicator: BaseIndicator{
			Period: period,
			Color:  color,
		},
	}
}

type rsi struct {
	BaseIndicator
	Values core.Series[float64]
}

// Warmup returns the number of candles needed to calculate the indicator
func (r rsi) Warmup() int {
	return r.Period
}

// Name returns the formatted name of the indicator
func (r rsi) Name() string {
	return fmt.Sprintf("RSI(%d)", r.Period)
}

// Overlay returns true if the indicator should be drawn on the price chart
func (r rsi) Overlay() bool {
	return false
}

// Load calculates the indicator values from the provided dataframe
func (r *rsi) Load(dataframe *core.Dataframe) {
	if !ValidateDataframe(dataframe, r.Period) {
		return
	}

	values := talib.Rsi(dataframe.Close, r.Period)
	r.Values, r.Time = TrimData(values, dataframe.Time, r.Period)
}

// Metrics returns the visual representation of the indicator
func (r rsi) Metrics() []plot.IndicatorMetric {
	return []plot.IndicatorMetric{
		CreateMetric("line", r.Color, r.Values, r.Time),
	}
}
