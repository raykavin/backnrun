package metric

import (
	"sort"

	"github.com/samber/lo"
	"gonum.org/v1/gonum/stat"
)

// BootstrapInterval represents the confidence interval calculated by the bootstrap method.
type BootstrapInterval struct {
	Lower  float64 // Lower bound of the confidence interval
	Upper  float64 // Upper bound of the confidence interval
	StdDev float64 // Standard deviation of the bootstrap samples
	Mean   float64 // Mean of the bootstrap samples
}

// Bootstrap calculates the confidence interval of a sample using the bootstrap method.
// Parameters:
//   - values: The original sample data
//   - measure: The statistical function to apply to each bootstrap sample
//   - sampleSize: Number of bootstrap samples to generate
//   - confidence: Confidence level (e.g., 0.95 for 95% confidence)
func Bootstrap(values []float64, measure func([]float64) float64, sampleSize int,
	confidence float64) BootstrapInterval {

	if len(values) == 0 {
		return BootstrapInterval{}
	}

	// Generate bootstrap samples and compute the measure for each
	data := generateBootstrapSamples(values, measure, sampleSize)

	// Calculate confidence interval
	tail := 1 - confidence
	sort.Float64s(data)

	mean, stdDev := stat.MeanStdDev(data, nil)
	upper := stat.Quantile(1-tail/2, stat.LinInterp, data, nil)
	lower := stat.Quantile(tail/2, stat.LinInterp, data, nil)

	return BootstrapInterval{
		Lower:  lower,
		Upper:  upper,
		StdDev: stdDev,
		Mean:   mean,
	}
}

// generateBootstrapSamples creates bootstrap samples and applies the measure function to each.
func generateBootstrapSamples(values []float64, measure func([]float64) float64, sampleSize int) []float64 {
	data := make([]float64, 0, sampleSize)

	for i := 0; i < sampleSize; i++ {
		// Create a bootstrap sample with replacement
		samples := make([]float64, len(values))
		for j := 0; j < len(values); j++ {
			samples[j] = lo.Sample(values)
		}

		// Apply the measure function to the bootstrap sample
		data = append(data, measure(samples))
	}

	return data
}
