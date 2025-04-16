package optimizer

import (
	"context"
	"fmt"
	"time"

	"github.com/raykavin/backnrun/pkg/logger"
)

// Parameter represents a configurable parameter that can be optimized
type Parameter struct {
	Name        string      // Name of the parameter
	Description string      // Description of what the parameter does
	Default     interface{} // Default value
	Min         interface{} // Minimum value (for numeric parameters)
	Max         interface{} // Maximum value (for numeric parameters)
	Step        interface{} // Step size (for numeric parameters in grid search)
	Options     []interface{} // Possible values (for categorical parameters)
	Type        ParameterType // Type of the parameter
}

// ParameterType defines the data type of a parameter
type ParameterType string

const (
	// TypeInt represents integer parameters
	TypeInt ParameterType = "int"
	// TypeFloat represents floating-point parameters
	TypeFloat ParameterType = "float"
	// TypeBool represents boolean parameters
	TypeBool ParameterType = "bool"
	// TypeString represents string parameters
	TypeString ParameterType = "string"
	// TypeCategorical represents categorical parameters with predefined options
	TypeCategorical ParameterType = "categorical"
)

// ParameterSet represents a collection of parameters with specific values
type ParameterSet map[string]interface{}

// Result represents the outcome of a single optimization run
type Result struct {
	Parameters ParameterSet     // The parameter values used
	Metrics    map[string]float64 // Performance metrics
	Duration   time.Duration    // How long the evaluation took
}

// MetricName defines standard metric names for optimization
type MetricName string

const (
	// MetricProfit represents the total profit
	MetricProfit MetricName = "profit"
	// MetricWinRate represents the percentage of winning trades
	MetricWinRate MetricName = "win_rate"
	// MetricPayoff represents the payoff ratio
	MetricPayoff MetricName = "payoff"
	// MetricProfitFactor represents the profit factor
	MetricProfitFactor MetricName = "profit_factor"
	// MetricSQN represents the System Quality Number
	MetricSQN MetricName = "sqn"
	// MetricDrawdown represents the maximum drawdown
	MetricDrawdown MetricName = "drawdown"
	// MetricSharpeRatio represents the Sharpe ratio
	MetricSharpeRatio MetricName = "sharpe_ratio"
	// MetricTradeCount represents the total number of trades
	MetricTradeCount MetricName = "trade_count"
)

// Evaluator defines the interface for evaluating a parameter set
type Evaluator interface {
	// Evaluate runs a backtest with the given parameters and returns performance metrics
	Evaluate(ctx context.Context, params ParameterSet) (*Result, error)
}

// Optimizer defines the interface for optimization algorithms
type Optimizer interface {
	// Optimize runs the optimization process and returns the best parameter sets
	Optimize(ctx context.Context, evaluator Evaluator, targetMetric MetricName, maximize bool) ([]*Result, error)
	// SetParameters sets the parameters to be optimized
	SetParameters(params []Parameter) error
	// SetMaxIterations sets the maximum number of iterations for the optimization
	SetMaxIterations(iterations int)
	// SetParallelism sets the number of parallel evaluations
	SetParallelism(n int)
}

// Config holds configuration for the optimization process
type Config struct {
	// Parameters to optimize
	Parameters []Parameter
	// Maximum number of iterations
	MaxIterations int
	// Number of parallel evaluations
	Parallelism int
	// Logger instance
	Logger logger.Logger
	// Target metric to optimize
	TargetMetric MetricName
	// Whether to maximize (true) or minimize (false) the target metric
	Maximize bool
	// Top N results to return
	TopN int
}

// NewConfig creates a default configuration
func NewConfig() *Config {
	return &Config{
		Parameters:    []Parameter{},
		MaxIterations: 100,
		Parallelism:   1,
		Logger:        nil,
		TargetMetric:  MetricProfit,
		Maximize:      true,
		TopN:          5,
	}
}

// WithParameters adds parameters to the configuration
func (c *Config) WithParameters(params ...Parameter) *Config {
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
func (c *Config) WithLogger(logger logger.Logger) *Config {
	c.Logger = logger
	return c
}

// WithTargetMetric sets the target metric to optimize
func (c *Config) WithTargetMetric(metric MetricName, maximize bool) *Config {
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
func ValidateParameterSet(params ParameterSet, definitions []Parameter) error {
	for _, def := range definitions {
		value, exists := params[def.Name]
		if !exists {
			return fmt.Errorf("missing parameter: %s", def.Name)
		}

		switch def.Type {
		case TypeInt:
			if _, ok := value.(int); !ok {
				return fmt.Errorf("parameter %s must be an integer", def.Name)
			}
		case TypeFloat:
			if _, ok := value.(float64); !ok {
				return fmt.Errorf("parameter %s must be a float", def.Name)
			}
		case TypeBool:
			if _, ok := value.(bool); !ok {
				return fmt.Errorf("parameter %s must be a boolean", def.Name)
			}
		case TypeString:
			if _, ok := value.(string); !ok {
				return fmt.Errorf("parameter %s must be a string", def.Name)
			}
		case TypeCategorical:
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

// StrategyEvaluator is an interface for strategies that can be optimized
type StrategyEvaluator interface {
	// GetParameters returns the parameters that can be optimized
	GetParameters() []Parameter
	// SetParameterValues sets parameter values for evaluation
	SetParameterValues(params ParameterSet) error
}

// ResultSorter sorts optimization results by a specific metric
type ResultSorter struct {
	Results      []*Result
	MetricName   string
	Maximize     bool
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
