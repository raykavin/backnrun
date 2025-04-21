package indicator

import (
	"fmt"
	"time"

	"github.com/raykavin/backnrun/core"
	"github.com/raykavin/backnrun/plot"
)

type SeriesType int8

const (
	Close SeriesType = iota
	Open
	High
	Low
)

func (s SeriesType) FromDataframe(dt *core.Dataframe) ([]float64, error) {
	seriesTypeMap := map[SeriesType][]float64{
		Close: dt.Close,
		High:  dt.High,
		Low:   dt.Low,
		Open:  dt.Low,
	}

	if sType, ok := seriesTypeMap[s]; ok {
		return sType, nil
	}

	return nil, fmt.Errorf("invalid series type")
}

// BaseIndicator provides common functionality for all indicators
type BaseIndicator struct {
	Period int
	Color  string
	Series SeriesType
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
