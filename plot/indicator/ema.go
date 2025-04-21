package indicator

import (
	"fmt"

	"github.com/raykavin/backnrun/core"
	"github.com/raykavin/backnrun/plot"

	"github.com/markcheno/go-talib"
)

// EMA creates a new Exponential Moving Average indicator
// period: the number of periods to use for calculations
// color: the color to use for the indicator line
func EMA(period int, color string, seriesType SeriesType) plot.Indicator {
	return &ema{
		BaseIndicator: BaseIndicator{
			Period: period,
			Color:  color,
			Series: seriesType,
		},
	}
}

type ema struct {
	BaseIndicator
	Values core.Series[float64]
}

// Warmup returns the number of candles needed to calculate the indicator
func (e ema) Warmup() int { return e.Period }

// Name returns the formatted name of the indicator
func (e ema) Name() string { return fmt.Sprintf("EMA(%d)", e.Period) }

// Overlay returns true if the indicator should be drawn on the price chart
func (e ema) Overlay() bool { return true }

// Metrics returns the visual representation of the indicator
func (e ema) Metrics() []plot.IndicatorMetric {
	return []plot.IndicatorMetric{
		CreateMetric("line", e.Color, e.Values, e.Time),
	}
}

// Load calculates the indicator values from the provided dataframe
func (e *ema) Load(df *core.Dataframe) {
	if !ValidateDataframe(df, e.Period) {
		return
	}

	seriesType, err := e.Series.FromDataframe(df)
	if err != nil {
		return
	}

	values := talib.Ema(seriesType, e.Period)
	e.Values, e.Time = TrimData(values, df.Time, e.Period)
}
