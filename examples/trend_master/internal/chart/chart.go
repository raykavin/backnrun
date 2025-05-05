package chart

import (
	"github.com/raykavin/backnrun/bot"
	"github.com/raykavin/backnrun/examples/trend_master/internal/strategy"
	"github.com/raykavin/backnrun/plot"
	"github.com/raykavin/backnrun/plot/indicator"
)

// Chart is an alias to the underlying chart implementation
type Chart = plot.Chart

// SetupChartServer creates and configures the chart server
func SetupChartServer(indicators strategy.IndicatorsConfig) (*plot.ChartServer, *plot.Chart, error) {
	// Extract indicator periods from configuration
	emaPeriods := []int{
		indicators.EMA.FastPeriod,
		indicators.EMA.SlowPeriod,
		indicators.EMA.LongPeriod,
	}

	// Create new chart with custom indicators
	chart, err := plot.NewChart(
		bot.DefaultLog,
		plot.WithCustomIndicators(
			indicator.EMA(emaPeriods[0], "lime", indicator.Close),
			indicator.EMA(emaPeriods[1], "red", indicator.Close),
			indicator.MACD(
				indicators.MACD.FastPeriod,
				indicators.MACD.SlowPeriod,
				indicators.MACD.SignalPeriod,
				"blue", "red", "green",
			),
		),
	)
	if err != nil {
		return nil, nil, err
	}

	// Create and return chart server
	return plot.NewChartServer(chart, plot.NewStandardHTTPServer(), bot.DefaultLog), chart, nil
}
