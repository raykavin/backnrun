package indicator

import (
	"time"

	"github.com/rodrigo-brito/ninjabot/model"
	"github.com/rodrigo-brito/ninjabot/plot"
)

// BaseIndicator provides common functionality for all indicators
type BaseIndicator struct {
	Period int
	Color  string
	Time   []time.Time
}

// CreateMetric creates a standard indicator metric
func CreateMetric(style, color string, values model.Series[float64], time []time.Time, name ...string) plot.IndicatorMetric {
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
func ValidateDataframe(dataframe *model.Dataframe, period int) bool {
	return len(dataframe.Time) >= period
}

// TrimData trims the data to match the period
func TrimData(data model.Series[float64], time []time.Time, period int) (model.Series[float64], []time.Time) {
	if period <= 0 || len(data) <= period {
		return data, time
	}
	return data[period:], time[period:]
}
