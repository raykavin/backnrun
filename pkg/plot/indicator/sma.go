package indicator

import (
	"fmt"

	"github.com/rodrigo-brito/ninjabot/model"
	"github.com/rodrigo-brito/ninjabot/plot"

	"github.com/markcheno/go-talib"
)

// SMA creates a new Simple Moving Average indicator
// period: the number of periods to use for calculations
// color: the color to use for the indicator line
func SMA(period int, color string) plot.Indicator {
	return &sma{
		BaseIndicator: BaseIndicator{
			Period: period,
			Color:  color,
		},
	}
}

type sma struct {
	BaseIndicator
	Values model.Series[float64]
}

// Warmup returns the number of candles needed to calculate the indicator
func (s sma) Warmup() int {
	return s.Period
}

// Name returns the formatted name of the indicator
func (s sma) Name() string {
	return fmt.Sprintf("SMA(%d)", s.Period)
}

// Overlay returns true if the indicator should be drawn on the price chart
func (s sma) Overlay() bool {
	return true
}

// Load calculates the indicator values from the provided dataframe
func (s *sma) Load(dataframe *model.Dataframe) {
	if !ValidateDataframe(dataframe, s.Period) {
		return
	}

	values := talib.Sma(dataframe.Close, s.Period)
	s.Values, s.Time = TrimData(values, dataframe.Time, s.Period)
}

// Metrics returns the visual representation of the indicator
func (s sma) Metrics() []plot.IndicatorMetric {
	return []plot.IndicatorMetric{
		CreateMetric("line", s.Color, s.Values, s.Time),
	}
}
