package indicator

import (
	"fmt"

	"github.com/rodrigo-brito/ninjabot/model"
	"github.com/rodrigo-brito/ninjabot/plot"

	"github.com/markcheno/go-talib"
)

// SuperTrend creates a new SuperTrend indicator
// period: the number of periods to use for ATR calculation
// factor: the multiplier for the ATR
// color: the color to use for the indicator line
func SuperTrend(period int, factor float64, color string) plot.Indicator {
	return &supertrend{
		BaseIndicator: BaseIndicator{
			Period: period,
			Color:  color,
		},
		Factor: factor,
	}
}

type supertrend struct {
	BaseIndicator
	Factor         float64
	SuperTrend     model.Series[float64]
}

// Warmup returns the number of candles needed to calculate the indicator
func (s supertrend) Warmup() int {
	return s.Period
}

// Name returns the formatted name of the indicator
func (s supertrend) Name() string {
	return fmt.Sprintf("SuperTrend(%d,%.1f)", s.Period, s.Factor)
}

// Overlay returns true if the indicator should be drawn on the price chart
func (s supertrend) Overlay() bool {
	return true
}

// calculateBands calculates the basic upper and lower bands
func calculateBands(high, low float64, atr, factor float64) (float64, float64) {
	median := (high + low) / 2.0
	upperBand := median + atr*factor
	lowerBand := median - atr*factor
	return upperBand, lowerBand
}

// updateFinalBands updates the final upper and lower bands based on previous values
func updateFinalBands(
	basicUpper, basicLower float64,
	prevFinalUpper, prevFinalLower float64,
	prevClose float64,
) (float64, float64) {
	finalUpper := basicUpper
	if basicUpper < prevFinalUpper || prevClose > prevFinalUpper {
		finalUpper = basicUpper
	} else {
		finalUpper = prevFinalUpper
	}

	finalLower := basicLower
	if basicLower > prevFinalLower || prevClose < prevFinalLower {
		finalLower = basicLower
	} else {
		finalLower = prevFinalLower
	}

	return finalUpper, finalLower
}

// determineTrend determines the SuperTrend value based on the bands and previous trend
func determineTrend(
	finalUpper, finalLower float64,
	currentClose float64,
	prevSuperTrend, prevFinalUpper float64,
) float64 {
	if prevFinalUpper == prevSuperTrend {
		// Previous trend was down
		if currentClose > finalUpper {
			return finalLower // Trend changed to up
		}
		return finalUpper // Trend remains down
	}
	
	// Previous trend was up
	if currentClose < finalLower {
		return finalUpper // Trend changed to down
	}
	return finalLower // Trend remains up
}

// Load calculates the indicator values from the provided dataframe
func (s *supertrend) Load(dataframe *model.Dataframe) {
	if !ValidateDataframe(dataframe, s.Period) {
		return
	}

	atr := talib.Atr(dataframe.High, dataframe.Low, dataframe.Close, s.Period)
	dataLength := len(atr)
	
	// Initialize arrays
	basicUpperBand := make([]float64, dataLength)
	basicLowerBand := make([]float64, dataLength)
	finalUpperBand := make([]float64, dataLength)
	finalLowerBand := make([]float64, dataLength)
	superTrend := make([]float64, dataLength)

	// Calculate initial bands for index 0
	basicUpperBand[0], basicLowerBand[0] = calculateBands(
		dataframe.High[0], dataframe.Low[0], atr[0], s.Factor,
	)
	finalUpperBand[0] = basicUpperBand[0]
	finalLowerBand[0] = basicLowerBand[0]
	
	// Initial trend is assumed to be down (upper band is the SuperTrend)
	superTrend[0] = finalUpperBand[0]

	// Calculate for the rest of the data points
	for i := 1; i < dataLength; i++ {
		// Calculate basic bands
		basicUpperBand[i], basicLowerBand[i] = calculateBands(
			dataframe.High[i], dataframe.Low[i], atr[i], s.Factor,
		)
		
		// Update final bands
		finalUpperBand[i], finalLowerBand[i] = updateFinalBands(
			basicUpperBand[i], basicLowerBand[i],
			finalUpperBand[i-1], finalLowerBand[i-1],
			dataframe.Close[i-1],
		)
		
		// Determine trend
		superTrend[i] = determineTrend(
			finalUpperBand[i], finalLowerBand[i],
			dataframe.Close[i],
			superTrend[i-1], finalUpperBand[i-1],
		)
	}

	// Trim data to match the period
	s.SuperTrend, s.Time = TrimData(superTrend, dataframe.Time, s.Period)
}

// Metrics returns the visual representation of the indicator
func (s supertrend) Metrics() []plot.IndicatorMetric {
	return []plot.IndicatorMetric{
		CreateMetric("scatter", s.Color, s.SuperTrend, s.Time),
	}
}
