package indicator

import "github.com/markcheno/go-talib"

// SuperTrend calculates the SuperTrend indicator based on high, low, and close prices
// Parameters:
//   - high: slice of high prices
//   - low: slice of low prices
//   - close: slice of closing prices
//   - atrPeriod: period for Average True Range calculation
//   - factor: multiplier for the ATR
// Returns: slice of SuperTrend values
func SuperTrend(high, low, close []float64, atrPeriod int, factor float64) []float64 {
	length := len(close)
	if length == 0 {
		return []float64{}
	}

	// Calculate Average True Range
	atr := talib.Atr(high, low, close, atrPeriod)

	// Initialize all required bands
	basicUpperBand := make([]float64, length)
	basicLowerBand := make([]float64, length)
	finalUpperBand := make([]float64, length)
	finalLowerBand := make([]float64, length)
	superTrend := make([]float64, length)

	// Skip first element since we need previous values
	for i := 1; i < length; i++ {
		// Calculate basic bands
		median := (high[i] + low[i]) / 2.0
		basicUpperBand[i] = median + atr[i]*factor
		basicLowerBand[i] = median - atr[i]*factor

		// Calculate final upper band
		if basicUpperBand[i] < finalUpperBand[i-1] || close[i-1] > finalUpperBand[i-1] {
			finalUpperBand[i] = basicUpperBand[i]
		} else {
			finalUpperBand[i] = finalUpperBand[i-1]
		}

		// Calculate final lower band
		if basicLowerBand[i] > finalLowerBand[i-1] || close[i-1] < finalLowerBand[i-1] {
			finalLowerBand[i] = basicLowerBand[i]
		} else {
			finalLowerBand[i] = finalLowerBand[i-1]
		}

		// Determine SuperTrend value based on previous SuperTrend and current price
		if finalUpperBand[i-1] == superTrend[i-1] {
			// Previous SuperTrend was the upper band
			if close[i] > finalUpperBand[i] {
				superTrend[i] = finalLowerBand[i] // Trend changed to up
			} else {
				superTrend[i] = finalUpperBand[i] // Trend remains down
			}
		} else {
			// Previous SuperTrend was the lower band
			if close[i] < finalLowerBand[i] {
				superTrend[i] = finalUpperBand[i] // Trend changed to down
			} else {
				superTrend[i] = finalLowerBand[i] // Trend remains up
			}
		}
	}

	return superTrend
}
