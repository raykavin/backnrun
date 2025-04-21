package strategy

import "github.com/raykavin/backnrun/core"

// DataframeManager handles operations related to updating and maintaining the dataframe
type DataframeManager struct {
	dataframe *core.Dataframe
}

// NewDataframeManager creates a new dataframe manager for a given trading pair
func NewDataframeManager(pair string) *DataframeManager {
	dataframe := &core.Dataframe{
		Pair:     pair,
		Metadata: make(map[string]core.Series[float64]),
	}

	return &DataframeManager{
		dataframe: dataframe,
	}
}

// GetDataframe returns the current dataframe
func (dm *DataframeManager) GetDataframe() *core.Dataframe {
	return dm.dataframe
}

// GetSample returns a sample of the dataframe based on the warmup period
func (dm *DataframeManager) GetSample(warmupPeriod int) core.Dataframe {
	return dm.dataframe.Sample(warmupPeriod)
}

// UpdateDataFrame updates the dataframe with a new candle
func (dm *DataframeManager) UpdateDataFrame(candle core.Candle) {
	if len(dm.dataframe.Time) > 0 && candle.Time.Equal(dm.dataframe.Time[len(dm.dataframe.Time)-1]) {
		// Update the last candle if it has the same timestamp
		last := len(dm.dataframe.Time) - 1
		dm.dataframe.Close[last] = candle.Close
		dm.dataframe.Open[last] = candle.Open
		dm.dataframe.High[last] = candle.High
		dm.dataframe.Low[last] = candle.Low
		dm.dataframe.Volume[last] = candle.Volume
		dm.dataframe.Time[last] = candle.Time
		for k, v := range candle.Metadata {
			dm.dataframe.Metadata[k][last] = v
		}
	} else {
		// Append a new candle
		dm.dataframe.Close = append(dm.dataframe.Close, candle.Close)
		dm.dataframe.Open = append(dm.dataframe.Open, candle.Open)
		dm.dataframe.High = append(dm.dataframe.High, candle.High)
		dm.dataframe.Low = append(dm.dataframe.Low, candle.Low)
		dm.dataframe.Volume = append(dm.dataframe.Volume, candle.Volume)
		dm.dataframe.Time = append(dm.dataframe.Time, candle.Time)
		dm.dataframe.LastUpdate = candle.Time
		for k, v := range candle.Metadata {
			dm.dataframe.Metadata[k] = append(dm.dataframe.Metadata[k], v)
		}
	}
}

// HasSufficientData checks if the dataframe has enough data based on the warmup period
func (dm *DataframeManager) HasSufficientData(warmupPeriod int) bool {
	return len(dm.dataframe.Close) >= warmupPeriod
}

// IsLateCandle checks if a candle is older than the latest one in the dataframe
func (dm *DataframeManager) IsLateCandle(candle core.Candle) bool {
	return len(dm.dataframe.Time) > 0 && candle.Time.Before(dm.dataframe.Time[len(dm.dataframe.Time)-1])
}
