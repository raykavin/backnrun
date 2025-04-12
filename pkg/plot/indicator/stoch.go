package indicator

import (
	"fmt"

	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/plot"

	"github.com/markcheno/go-talib"
)

// Stoch creates a new Stochastic Oscillator indicator
// fastK: the fast %K period
// slowK: the slow %K period
// slowD: the slow %D period
// colorK: color for the %K line
// colorD: color for the %D line
func Stoch(fastK, slowK, slowD int, colorK, colorD string) plot.Indicator {
	return &stoch{
		FastK:  fastK,
		SlowK:  slowK,
		SlowD:  slowD,
		ColorK: colorK,
		ColorD: colorD,
	}
}

type stoch struct {
	BaseIndicator
	FastK   int
	SlowK   int
	SlowD   int
	ColorK  string
	ColorD  string
	ValuesK core.Series[float64]
	ValuesD core.Series[float64]
}

// Warmup returns the number of candles needed to calculate the indicator
func (s stoch) Warmup() int {
	return s.SlowD + s.SlowK
}

// Name returns the formatted name of the indicator
func (s stoch) Name() string {
	return fmt.Sprintf("STOCH(%d, %d, %d)", s.FastK, s.SlowK, s.SlowD)
}

// Overlay returns true if the indicator should be drawn on the price chart
func (s stoch) Overlay() bool {
	return false
}

// Load calculates the indicator values from the provided dataframe
func (s *stoch) Load(dataframe *core.Dataframe) {
	warmup := s.Warmup()
	if !ValidateDataframe(dataframe, warmup) {
		return
	}

	k, d := talib.Stoch(
		dataframe.High, dataframe.Low, dataframe.Close, s.FastK, s.SlowK, talib.SMA, s.SlowD, talib.SMA,
	)

	s.ValuesK, s.Time = TrimData(k, dataframe.Time, warmup)
	s.ValuesD, _ = TrimData(d, dataframe.Time, warmup)
}

// Metrics returns the visual representation of the indicator
func (s stoch) Metrics() []plot.IndicatorMetric {
	return []plot.IndicatorMetric{
		CreateMetric("line", s.ColorK, s.ValuesK, s.Time, "K"),
		CreateMetric("line", s.ColorD, s.ValuesD, s.Time, "D"),
	}
}
