package core

import (
	"fmt"
	"strconv"
	"time"
)

type CandleSubscriber interface {
	OnCandle(Candle)
}

// Candle represents a trading candle with OHLCV data
type Candle struct {
	Pair      string
	Time      time.Time
	UpdatedAt time.Time
	Open      float64
	Close     float64
	Low       float64
	High      float64
	Volume    float64
	Complete  bool

	// Additional columns from CSV inputs
	Metadata map[string]float64
}

// GetPair returns the trading pair identifier for the candle
func (c Candle) GetPair() string { return c.Pair }

// GetTime returns the timestamp of the candle
func (c Candle) GetTime() time.Time { return c.Time }

// GetUpdatedAt returns the last update time of the candle
func (c Candle) GetUpdatedAt() time.Time { return c.UpdatedAt }

// GetOpen returns the opening price of the candle
func (c Candle) GetOpen() float64 { return c.Open }

// GetClose returns the closing price of the candle
func (c Candle) GetClose() float64 { return c.Close }

// GetLow returns the lowest price during the candle period
func (c Candle) GetLow() float64 { return c.Low }

// GetHigh returns the highest price during the candle period
func (c Candle) GetHigh() float64 { return c.High }

// GetVolume returns the trading volume during the candle period
func (c Candle) GetVolume() float64 { return c.Volume }

// IsComplete returns whether the candle period is complete
func (c Candle) IsComplete() bool { return c.Complete }

// GetMetadata returns the additional metadata associated with the candle
func (c Candle) GetMetadata() map[string]float64 { return c.Metadata }

// Empty checks if the candle contains no significant data
func (c Candle) IsEmpty() bool { return c.Pair == "" && c.Close == 0 && c.Open == 0 && c.Volume == 0 }

// ToSlice converts a candle to a string slice for serialization
// with the specified decimal precision
func (c Candle) ToSlice(precision int) []string {
	return []string{
		fmt.Sprintf("%d", c.Time.Unix()),
		strconv.FormatFloat(c.Open, 'f', precision, 64),
		strconv.FormatFloat(c.Close, 'f', precision, 64),
		strconv.FormatFloat(c.Low, 'f', precision, 64),
		strconv.FormatFloat(c.High, 'f', precision, 64),
		strconv.FormatFloat(c.Volume, 'f', precision, 64),
	}
}

// Less implements the Item interface for comparison in priority queue
func (c Candle) Less(j Item) bool {
	other := j.(Candle)

	// Primary sort by time
	diff := other.Time.Sub(c.Time)
	if diff != 0 {
		return diff > 0
	}

	// Secondary sort by update time
	diff = other.UpdatedAt.Sub(c.UpdatedAt)
	if diff != 0 {
		return diff > 0
	}

	// Tertiary sort by pair name
	return c.Pair < other.Pair
}

// ToHeikinAshi transforms a regular candle into a Heikin-Ashi candle
func (c Candle) ToHeikinAshi(ha *HeikinAshi) Candle {
	haCandle := ha.CalculateHeikinAshi(c)

	return Candle{
		Pair:      c.Pair,
		Open:      haCandle.Open,
		High:      haCandle.High,
		Low:       haCandle.Low,
		Close:     haCandle.Close,
		Volume:    c.Volume,
		Complete:  c.Complete,
		Time:      c.Time,
		UpdatedAt: c.UpdatedAt,
	}
}
