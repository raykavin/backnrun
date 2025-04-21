package optimizer

import (
	"fmt"

	"github.com/raykavin/backnrun/core"
)

// Config holds configuration for the optimization process
type Config struct {
	// Parameters to optimize
	Parameters []core.Parameter
	// Maximum number of iterations
	MaxIterations int
	// Number of parallel evaluations
	Parallelism int
	// Logger instance
	Logger core.Logger
	// Target metric to optimize
	TargetMetric core.MetricName
	// Whether to maximize (true) or minimize (false) the target metric
	Maximize bool
	// Top N results to return
	TopN int
}

// NewConfig creates a default configuration
func NewConfig() *Config {
	return &Config{
		Parameters:    []core.Parameter{},
		MaxIterations: 100,
		Parallelism:   1,
		TargetMetric:  core.MetricProfit,
		Maximize:      true,
		TopN:          5,
		Logger:        nil,
	}
}

// WithParameters adds parameters to the configuration
func (c *Config) WithParameters(params ...core.Parameter) *Config {
	c.Parameters = append(c.Parameters, params...)
	return c
}

// WithMaxIterations sets the maximum number of iterations
func (c *Config) WithMaxIterations(iterations int) *Config {
	c.MaxIterations = iterations
	return c
}

// WithParallelism sets the number of parallel evaluations
func (c *Config) WithParallelism(n int) *Config {
	c.Parallelism = n
	return c
}

// WithLogger sets the logger
func (c *Config) WithLogger(logger core.Logger) *Config {
	c.Logger = logger
	return c
}

// WithTargetMetric sets the target metric to optimize
func (c *Config) WithTargetMetric(metric core.MetricName, maximize bool) *Config {
	c.TargetMetric = metric
	c.Maximize = maximize
	return c
}

// WithTopN sets the number of top results to return
func (c *Config) WithTopN(n int) *Config {
	c.TopN = n
	return c
}

// ValidateParameterSet checks if a parameter set contains all required parameters
// with values of the correct type
func ValidateParameterSet(params core.ParameterSet, definitions []core.Parameter) error {
	for _, def := range definitions {
		value, exists := params[def.Name]
		if !exists {
			return fmt.Errorf("missing parameter: %s", def.Name)
		}

		switch def.Type {
		case core.TypeInt:
			if _, ok := value.(int); !ok {
				return fmt.Errorf("parameter %s must be an integer", def.Name)
			}
		case core.TypeFloat:
			if _, ok := value.(float64); !ok {
				return fmt.Errorf("parameter %s must be a float", def.Name)
			}
		case core.TypeBool:
			if _, ok := value.(bool); !ok {
				return fmt.Errorf("parameter %s must be a boolean", def.Name)
			}
		case core.TypeString:
			if _, ok := value.(string); !ok {
				return fmt.Errorf("parameter %s must be a string", def.Name)
			}
		case core.TypeCategorical:
			found := false
			for _, option := range def.Options {
				if option == value {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("parameter %s has invalid value", def.Name)
			}
		}
	}
	return nil
}

// ResultSorter sorts optimization results by a specific metric
type ResultSorter struct {
	Results    []*core.OptimizerResult
	MetricName string
	Maximize   bool
}

// Len returns the number of results
func (s ResultSorter) Len() int {
	return len(s.Results)
}

// Swap swaps two results
func (s ResultSorter) Swap(i, j int) {
	s.Results[i], s.Results[j] = s.Results[j], s.Results[i]
}

// Less compares two results based on the target metric
func (s ResultSorter) Less(i, j int) bool {
	valueI := s.Results[i].Metrics[s.MetricName]
	valueJ := s.Results[j].Metrics[s.MetricName]

	if s.Maximize {
		return valueI > valueJ
	}
	return valueI < valueJ
}
