package core

import (
	"math"
)

// HeikinAshi handles the calculation of Heikin-Ashi candles
// Heikin-Ashi is a candlestick charting technique that filters out market noise
type HeikinAshi struct {
	PreviousHACandle Candle
}

// NewHeikinAshi creates a new HeikinAshi calculator
func NewHeikinAshi() *HeikinAshi {
	return &HeikinAshi{}
}

// CalculateHeikinAshi transforms a standard candle into a Heikin-Ashi candle
// Formula:
// - HA_Close = (Open + High + Low + Close) / 4
// - HA_Open = (Previous HA_Open + Previous HA_Close) / 2
// - HA_High = Max(High, HA_Open, HA_Close)
// - HA_Low = Min(Low, HA_Open, HA_Close)
func (ha *HeikinAshi) CalculateHeikinAshi(c Candle) Candle {
	var hkCandle Candle

	openValue := ha.PreviousHACandle.Open
	closeValue := ha.PreviousHACandle.Close

	// First HA candle is calculated using current candle
	if ha.PreviousHACandle.IsEmpty() {
		openValue = c.Open
		closeValue = c.Close
	}

	hkCandle.Open = (openValue + closeValue) / 2
	hkCandle.Close = (c.Open + c.High + c.Low + c.Close) / 4
	hkCandle.High = math.Max(c.High, math.Max(hkCandle.Open, hkCandle.Close))
	hkCandle.Low = math.Min(c.Low, math.Min(hkCandle.Open, hkCandle.Close))

	// Save as previous for next calculation
	ha.PreviousHACandle = hkCandle

	return hkCandle
}
