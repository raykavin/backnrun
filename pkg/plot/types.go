package plot

import (
	"time"

	"github.com/raykavin/backnrun/pkg/core"
)

// Candle represents OHLCV data with associated orders
type Candle struct {
	Time   time.Time    `json:"time"`
	Open   float64      `json:"open"`
	Close  float64      `json:"close"`
	High   float64      `json:"high"`
	Low    float64      `json:"low"`
	Volume float64      `json:"volume"`
	Orders []core.Order `json:"orders"`
}

// Shape represents a visual shape on the chart
type Shape struct {
	StartX time.Time `json:"x0"`
	EndX   time.Time `json:"x1"`
	StartY float64   `json:"y0"`
	EndY   float64   `json:"y1"`
	Color  string    `json:"color"`
}

// AssetValue represents a point in time value of an asset
type AssetValue struct {
	Time  time.Time `json:"time"`
	Value float64   `json:"value"`
}

// IndicatorMetric represents a single metric within an indicator
type IndicatorMetric struct {
	Name   string
	Color  string
	Style  string
	Values core.Series[float64]
	Time   []time.Time
}

// Indicator interface defines the methods required to implement a chart indicator
type Indicator interface {
	Name() string
	Overlay() bool
	Warmup() int
	Metrics() []IndicatorMetric
	Load(dataframe *core.Dataframe)
}

// indicatorMetric is the JSON serializable version of IndicatorMetric
type indicatorMetric struct {
	Name   string      `json:"name"`
	Time   []time.Time `json:"time"`
	Values []float64   `json:"value"`
	Color  string      `json:"color"`
	Style  string      `json:"style"`
}

// plotIndicator is the JSON serializable version of an Indicator
type plotIndicator struct {
	Name    string            `json:"name"`
	Overlay bool              `json:"overlay"`
	Metrics []indicatorMetric `json:"metrics"`
	Warmup  int               `json:"-"`
}

// drawdown represents maximum drawdown information
type drawdown struct {
	Value string    `json:"value"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}
