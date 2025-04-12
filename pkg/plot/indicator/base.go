package indicator

import (
	"time"

	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/plot"
)

// BaseIndicator provides common functionality for all indicators
type BaseIndicator struct {
	Period int
	Color  string
	Time   []time.Time
}

// CreateMetric creates a standard indicator metric
func CreateMetric(style, color string, values core.Series[float64], time []time.Time, name ...string) plot.IndicatorMetric {
	metric := plot.IndicatorMetric{
		Style:  style,
		Color:  color,
		Values: values,
		Time:   time,
	}

	if len(name) > 0 {
		metric.Name = name[0]
	}

	return metric
}

// ValidateDataframe checks if the dataframe has enough data points for the indicator period
func ValidateDataframe(dataframe *core.Dataframe, period int) bool {
	return len(dataframe.Time) >= period
}

// TrimData trims the data to match the period
func TrimData(data core.Series[float64], time []time.Time, period int) (core.Series[float64], []time.Time) {
	if period <= 0 || len(data) <= period {
		return data, time
	}
	return data[period:], time[period:]
}
