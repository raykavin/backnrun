package optimizer

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/raykavin/backnrun/pkg/logger"
)

// RandomSearch implements a random search optimization algorithm
type RandomSearch struct {
	parameters    []Parameter
	maxIterations int
	parallelism   int
	logger        logger.Logger
	rng           *rand.Rand
}

// NewRandomSearch creates a new random search optimizer
func NewRandomSearch(config *Config) (*RandomSearch, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if len(config.Parameters) == 0 {
		return nil, fmt.Errorf("at least one parameter must be provided")
	}

	// Create a new random number generator with a time-based seed
	source := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(source)

	return &RandomSearch{
		parameters:    config.Parameters,
		maxIterations: config.MaxIterations,
		parallelism:   config.Parallelism,
		logger:        config.Logger,
		rng:           rng,
	}, nil
}

// SetParameters sets the parameters to be optimized
func (r *RandomSearch) SetParameters(params []Parameter) error {
	if len(params) == 0 {
		return fmt.Errorf("at least one parameter must be provided")
	}
	r.parameters = params
	return nil
}

// SetMaxIterations sets the maximum number of iterations
func (r *RandomSearch) SetMaxIterations(iterations int) {
	r.maxIterations = iterations
}

// SetParallelism sets the number of parallel evaluations
func (r *RandomSearch) SetParallelism(n int) {
	r.parallelism = n
}

// Optimize runs the random search optimization process
func (r *RandomSearch) Optimize(
	ctx context.Context,
	evaluator Evaluator,
	targetMetric MetricName,
	maximize bool,
) ([]*Result, error) {
	if evaluator == nil {
		return nil, fmt.Errorf("evaluator cannot be nil")
	}

	// Generate random parameter sets
	parameterSets := r.generateRandomParameterSets()

	r.logf("Starting random search with %d iterations", len(parameterSets))

	// Run evaluations
	results, err := r.runEvaluations(ctx, evaluator, parameterSets)
	if err != nil {
		return nil, err
	}

	// Sort results by the target metric
	sorter := ResultSorter{
		Results:    results,
		MetricName: string(targetMetric),
		Maximize:   maximize,
	}
	sort.Sort(sorter)

	r.logf("Random search completed with %d results", len(results))
	return results, nil
}

// generateRandomParameterSets creates random parameter sets for evaluation
func (r *RandomSearch) generateRandomParameterSets() []ParameterSet {
	parameterSets := make([]ParameterSet, r.maxIterations)

	for i := 0; i < r.maxIterations; i++ {
		paramSet := make(ParameterSet)
		for _, param := range r.parameters {
			value := r.generateRandomValue(param)
			paramSet[param.Name] = value
		}
		parameterSets[i] = paramSet
	}

	return parameterSets
}

// generateRandomValue creates a random value for a parameter based on its type and range
func (r *RandomSearch) generateRandomValue(param Parameter) interface{} {
	switch param.Type {
	case TypeInt:
		return r.generateRandomInt(param)
	case TypeFloat:
		return r.generateRandomFloat(param)
	case TypeBool:
		return r.rng.Intn(2) == 1
	case TypeString, TypeCategorical:
		return r.generateRandomOption(param)
	default:
		// Default to the parameter's default value if type is unsupported
		return param.Default
	}
}

// generateRandomInt creates a random integer within the specified range
func (r *RandomSearch) generateRandomInt(param Parameter) int {
	min, ok := param.Min.(int)
	if !ok {
		// Fall back to default if min is not an int
		if def, ok := param.Default.(int); ok {
			return def
		}
		return 0
	}

	max, ok := param.Max.(int)
	if !ok {
		// Fall back to min if max is not an int
		return min
	}

	if min >= max {
		return min
	}

	return min + r.rng.Intn(max-min+1)
}

// generateRandomFloat creates a random float within the specified range
func (r *RandomSearch) generateRandomFloat(param Parameter) float64 {
	min, ok := param.Min.(float64)
	if !ok {
		// Fall back to default if min is not a float
		if def, ok := param.Default.(float64); ok {
			return def
		}
		return 0.0
	}

	max, ok := param.Max.(float64)
	if !ok {
		// Fall back to min if max is not a float
		return min
	}

	if min >= max {
		return min
	}

	return min + r.rng.Float64()*(max-min)
}

// generateRandomOption selects a random option from the available options
func (r *RandomSearch) generateRandomOption(param Parameter) interface{} {
	if len(param.Options) == 0 {
		// Fall back to default if no options are available
		return param.Default
	}

	index := r.rng.Intn(len(param.Options))
	return param.Options[index]
}

// runEvaluations executes the evaluations for all parameter sets
func (r *RandomSearch) runEvaluations(
	ctx context.Context,
	evaluator Evaluator,
	parameterSets []ParameterSet,
) ([]*Result, error) {
	var (
		results   []*Result
		mutex     sync.Mutex
		wg        sync.WaitGroup
		errCh     = make(chan error, 1)
		semaphore = make(chan struct{}, r.parallelism)
	)

	// Process each parameter set
	for i, params := range parameterSets {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		case err := <-errCh:
			return results, err
		default:
			// Continue processing
		}

		wg.Add(1)
		semaphore <- struct{}{} // Acquire semaphore

		// Run evaluation in a goroutine
		go func(index int, paramSet ParameterSet) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore

			r.logf("Evaluating parameter set %d/%d", index+1, len(parameterSets))

			// Evaluate the parameter set
			result, err := evaluator.Evaluate(ctx, paramSet)
			if err != nil {
				select {
				case errCh <- fmt.Errorf("evaluation error: %w", err):
				default:
					// Another error was already sent
				}
				return
			}

			// Store the result
			mutex.Lock()
			results = append(results, result)
			mutex.Unlock()

			r.logf("Completed evaluation %d/%d", index+1, len(parameterSets))
		}(i, params)
	}

	// Wait for all evaluations to complete
	wg.Wait()

	// Check if there was an error
	select {
	case err := <-errCh:
		return results, err
	default:
		return results, nil
	}
}

// logf logs a message if a logger is configured
func (r *RandomSearch) logf(format string, args ...interface{}) {
	if r.logger != nil {
		r.logger.Infof(format, args...)
	}
}
