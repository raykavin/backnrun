package indicator

import (
	"fmt"

	"github.com/markcheno/go-talib"
	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/plot"
)

// CCI creates a new Commodity Channel Index indicator
// period: the number of periods to use for calculations
// color: the color to use for the indicator line
func CCI(period int, color string) plot.Indicator {
	return &cci{
		BaseIndicator: BaseIndicator{
			Period: period,
			Color:  color,
		},
	}
}

type cci struct {
	BaseIndicator
	Values core.Series[float64]
}

// Warmup returns the number of candles needed to calculate the indicator
func (c cci) Warmup() int {
	return c.Period
}

// Name returns the formatted name of the indicator
func (c cci) Name() string {
	return fmt.Sprintf("CCI(%d)", c.Period)
}

// Overlay returns true if the indicator should be drawn on the price chart
func (c cci) Overlay() bool {
	return false
}

// Load calculates the indicator values from the provided dataframe
func (c *cci) Load(dataframe *core.Dataframe) {
	if !ValidateDataframe(dataframe, c.Period) {
		return
	}
	
	values := talib.Cci(dataframe.High, dataframe.Low, dataframe.Close, c.Period)
	c.Values, c.Time = TrimData(values, dataframe.Time, c.Period)
}

// Metrics returns the visual representation of the indicator
func (c cci) Metrics() []plot.IndicatorMetric {
	return []plot.IndicatorMetric{
		CreateMetric("line", c.Color, c.Values, c.Time),
	}
}
