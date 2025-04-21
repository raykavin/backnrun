package core

import (
	"context"
	"time"
)

// Evaluator defines the interface for evaluating a parameter set
type Evaluator interface {
	// Evaluate runs a backtest with the given parameters and returns performance metrics
	Evaluate(ctx context.Context, params ParameterSet) (*OptimizerResult, error)
}

// OptimizableStrategy is an interface for strategies that can be optimized
type OptimizableStrategy interface {
	Strategy
	StrategyEvaluator
}

// Optimizer defines the interface for optimization algorithms
type Optimizer interface {
	// Optimize runs the optimization process and returns the best parameter sets
	Optimize(ctx context.Context, evaluator Evaluator, targetMetric MetricName, maximize bool) ([]*OptimizerResult, error)
	// SetParameters sets the parameters to be optimized
	SetParameters(params []Parameter) error
	// SetMaxIterations sets the maximum number of iterations for the optimization
	SetMaxIterations(iterations int)
	// SetParallelism sets the number of parallel evaluations
	SetParallelism(n int)
}

// StrategyEvaluator is an interface for strategies that can be optimized
type StrategyEvaluator interface {
	// GetParameters returns the parameters that can be optimized
	GetParameters() []Parameter
	// SetParameterValues sets parameter values for evaluation
	SetParameterValues(params ParameterSet) error
}

// ParameterSet represents a collection of parameters with specific values
type ParameterSet map[string]any

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

// Parameter represents a configurable parameter that can be optimized
type Parameter struct {
	Name        string        // Name of the parameter
	Description string        // Description of what the parameter does
	Default     any           // Default value
	Min         any           // Minimum value (for numeric parameters)
	Max         any           // Maximum value (for numeric parameters)
	Step        any           // Step size (for numeric parameters in grid search)
	Options     []any         // Possible values (for categorical parameters)
	Type        ParameterType // Type of the parameter
}

// Result represents the outcome of a single optimization run
type OptimizerResult struct {
	Parameters ParameterSet       // The parameter values used
	Metrics    map[string]float64 // Performance metrics
	Duration   time.Duration      // How long the evaluation took
}
