package indicator

import (
	"fmt"

	"github.com/raykavin/backnrun/core"
	"github.com/raykavin/backnrun/plot"

	"github.com/markcheno/go-talib"
)

// WillR creates a new Williams %R indicator
// period: the number of periods to use for calculations
// color: the color to use for the indicator line
func WillR(period int, color string) plot.Indicator {
	return &willR{
		BaseIndicator: BaseIndicator{
			Period: period,
			Color:  color,
		},
	}
}

type willR struct {
	BaseIndicator
	Values core.Series[float64]
}

// Warmup returns the number of candles needed to calculate the indicator
func (w willR) Warmup() int {
	return w.Period
}

// Name returns the formatted name of the indicator
func (w willR) Name() string {
	return fmt.Sprintf("%%R(%d)", w.Period)
}

// Overlay returns true if the indicator should be drawn on the price chart
func (w willR) Overlay() bool {
	return false
}

// Load calculates the indicator values from the provided dataframe
func (w *willR) Load(dataframe *core.Dataframe) {
	if !ValidateDataframe(dataframe, w.Period) {
		return
	}

	values := talib.WillR(dataframe.High, dataframe.Low, dataframe.Close, w.Period)
	w.Values, w.Time = TrimData(values, dataframe.Time, w.Period)
}

// Metrics returns the visual representation of the indicator
func (w willR) Metrics() []plot.IndicatorMetric {
	return []plot.IndicatorMetric{
		CreateMetric("line", w.Color, w.Values, w.Time),
	}
}
