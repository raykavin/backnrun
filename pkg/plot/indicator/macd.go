package indicator

import (
	"fmt"

	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/plot"

	"github.com/markcheno/go-talib"
)

// MACD creates a new Moving Average Convergence Divergence indicator
// fast: the fast period
// slow: the slow period
// signal: the signal period
// colorMACD: color for the MACD line
// colorMACDSignal: color for the signal line
// colorMACDHist: color for the histogram
func MACD(fast, slow, signal int, colorMACD, colorMACDSignal, colorMACDHist string) plot.Indicator {
	return &macd{
		Fast:            fast,
		Slow:            slow,
		Signal:          signal,
		ColorMACD:       colorMACD,
		ColorMACDSignal: colorMACDSignal,
		ColorMACDHist:   colorMACDHist,
	}
}

type macd struct {
	BaseIndicator
	Fast             int
	Slow             int
	Signal           int
	ColorMACD        string
	ColorMACDSignal  string
	ColorMACDHist    string
	ValuesMACD       core.Series[float64]
	ValuesMACDSignal core.Series[float64]
	ValuesMACDHist   core.Series[float64]
}

// Warmup returns the number of candles needed to calculate the indicator
func (m macd) Warmup() int {
	return m.Slow + m.Signal
}

// Name returns the formatted name of the indicator
func (m macd) Name() string {
	return fmt.Sprintf("MACD(%d, %d, %d)", m.Fast, m.Slow, m.Signal)
}

// Overlay returns true if the indicator should be drawn on the price chart
func (m macd) Overlay() bool {
	return false
}

// Load calculates the indicator values from the provided dataframe
func (m *macd) Load(dataframe *core.Dataframe) {
	warmup := m.Warmup()
	if !ValidateDataframe(dataframe, warmup) {
		return
	}

	macdLine, signalLine, histogram := talib.Macd(dataframe.Close, m.Fast, m.Slow, m.Signal)
	m.ValuesMACD, m.Time = TrimData(macdLine, dataframe.Time, warmup)
	m.ValuesMACDSignal, _ = TrimData(signalLine, dataframe.Time, warmup)
	m.ValuesMACDHist, _ = TrimData(histogram, dataframe.Time, warmup)
}

// Metrics returns the visual representation of the indicator
func (m macd) Metrics() []plot.IndicatorMetric {
	return []plot.IndicatorMetric{
		CreateMetric("line", m.ColorMACD, m.ValuesMACD, m.Time, "MACD"),
		CreateMetric("line", m.ColorMACDSignal, m.ValuesMACDSignal, m.Time, "MACDSignal"),
		CreateMetric("bar", m.ColorMACDHist, m.ValuesMACDHist, m.Time, "MACDHist"),
	}
}
