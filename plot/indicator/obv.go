package indicator

import (
	"github.com/raykavin/backnrun/core"
	"github.com/raykavin/backnrun/plot"

	"github.com/markcheno/go-talib"
)

// OBV creates a new On-Balance Volume indicator
// color: the color to use for the indicator line
func OBV(color string) plot.Indicator {
	return &obv{
		BaseIndicator: BaseIndicator{
			Period: 0,
			Color:  color,
		},
	}
}

type obv struct {
	BaseIndicator
	Values core.Series[float64]
}

// Warmup returns the number of candles needed to calculate the indicator
func (o obv) Warmup() int {
	return 0
}

// Name returns the name of the indicator
func (o obv) Name() string {
	return "OBV"
}

// Overlay returns true if the indicator should be drawn on the price chart
func (o obv) Overlay() bool {
	return false
}

// Load calculates the indicator values from the provided dataframe
func (o *obv) Load(dataframe *core.Dataframe) {
	o.Values = talib.Obv(dataframe.Close, dataframe.Volume)
	o.Time = dataframe.Time
}

// Metrics returns the visual representation of the indicator
func (o obv) Metrics() []plot.IndicatorMetric {
	return []plot.IndicatorMetric{
		CreateMetric("line", o.Color, o.Values, o.Time),
	}
}
