package core

import (
	"time"
)

// Dataframe is a time series container for OHLCV and custom indicator data
type Dataframe struct {
	Pair string

	Close  Series[float64]
	Open   Series[float64]
	High   Series[float64]
	Low    Series[float64]
	Volume Series[float64]

	Time       []time.Time
	LastUpdate time.Time

	// Custom user metadata for indicators
	Metadata map[string]Series[float64]
}

// Sample returns a subset of the dataframe with the last 'positions' elements
// Used for windowing operations on a dataframe
func (df Dataframe) Sample(positions int) Dataframe {
	size := len(df.Time)
	start := size - positions

	// Return the entire dataframe if requested sample is larger than dataframe
	if start <= 0 {
		return df
	}

	sample := Dataframe{
		Pair:       df.Pair,
		Close:      df.Close.LastValues(positions),
		Open:       df.Open.LastValues(positions),
		High:       df.High.LastValues(positions),
		Low:        df.Low.LastValues(positions),
		Volume:     df.Volume.LastValues(positions),
		Time:       df.Time[start:],
		LastUpdate: df.LastUpdate,
		Metadata:   make(map[string]Series[float64]),
	}

	// Also copy metadata series
	for key := range df.Metadata {
		sample.Metadata[key] = df.Metadata[key].LastValues(positions)
	}

	return sample
}
