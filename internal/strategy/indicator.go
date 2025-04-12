package strategy

import (
	"time"

	"github.com/raykavin/backnrun/internal/core"
)

type MetricStyle string

const (
	StyleBar       = "bar"
	StyleScatter   = "scatter"
	StyleLine      = "line"
	StyleHistogram = "histogram"
	StyleWaterfall = "waterfall"
)

type IndicatorMetric struct {
	Name   string
	Color  string
	Style  MetricStyle // default: line
	Values core.Series[float64]
}

type ChartIndicator struct {
	Time      []time.Time
	Metrics   []IndicatorMetric
	Overlay   bool
	GroupName string
	Warmup    int
}
