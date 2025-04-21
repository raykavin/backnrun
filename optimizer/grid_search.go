package optimizer

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/raykavin/backnrun/core"
)

// GridSearch implements a grid search optimization algorithm
type GridSearch struct {
	parameters    []core.Parameter
	maxIterations int
	parallelism   int
	log           core.Logger
}

// NewGridSearch creates a new grid search optimizer
func NewGridSearch(config *Config) (*GridSearch, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if len(config.Parameters) == 0 {
		return nil, fmt.Errorf("at least one parameter must be provided")
	}

	return &GridSearch{
		parameters:    config.Parameters,
		maxIterations: config.MaxIterations,
		parallelism:   config.Parallelism,
		log:           config.Logger,
	}, nil
}

// SetParameters sets the parameters to be optimized
func (g *GridSearch) SetParameters(params []core.Parameter) error {
	if len(params) == 0 {
		return fmt.Errorf("at least one parameter must be provided")
	}
	g.parameters = params
	return nil
}

// SetMaxIterations sets the maximum number of iterations
func (g *GridSearch) SetMaxIterations(iterations int) {
	g.maxIterations = iterations
}

// SetParallelism sets the number of parallel evaluations
func (g *GridSearch) SetParallelism(n int) {
	g.parallelism = n
}

// Optimize runs the grid search optimization process
func (g *GridSearch) Optimize(ctx context.Context, evaluator core.Evaluator, targetMetric core.MetricName, maximize bool) ([]*core.OptimizerResult, error) {
	if evaluator == nil {
		return nil, fmt.Errorf("evaluator cannot be nil")
	}

	// Generate all parameter combinations
	parameterSets, err := g.generateParameterSets()
	if err != nil {
		return nil, err
	}

	// Limit the number of combinations if it exceeds maxIterations
	if len(parameterSets) > g.maxIterations {
		g.logf("Limiting parameter combinations from %d to %d", len(parameterSets), g.maxIterations)
		parameterSets = parameterSets[:g.maxIterations]
	}

	g.logf("Starting grid search with %d parameter combinations", len(parameterSets))

	// Run evaluations
	results, err := g.runEvaluations(ctx, evaluator, parameterSets)
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

	g.logf("Grid search completed with %d results", len(results))
	return results, nil
}

// generateParameterSets creates all possible combinations of parameter values
func (g *GridSearch) generateParameterSets() ([]core.ParameterSet, error) {
	// Initialize with an empty parameter set
	parameterSets := []core.ParameterSet{make(core.ParameterSet)}

	// For each parameter, generate all possible values and create combinations
	for _, param := range g.parameters {
		values, err := g.generateParameterValues(param)
		if err != nil {
			return nil, err
		}

		// Create new combinations by adding each value to each existing set
		var newSets []core.ParameterSet
		for _, set := range parameterSets {
			for _, value := range values {
				// Create a new set with the current value
				newSet := make(core.ParameterSet)
				for k, v := range set {
					newSet[k] = v
				}
				newSet[param.Name] = value
				newSets = append(newSets, newSet)
			}
		}
		parameterSets = newSets
	}

	return parameterSets, nil
}

// generateParameterValues creates all possible values for a parameter based on its type and range
func (g *GridSearch) generateParameterValues(param core.Parameter) ([]any, error) {
	switch param.Type {
	case core.TypeInt:
		return g.generateIntValues(param)
	case core.TypeFloat:
		return g.generateFloatValues(param)
	case core.TypeBool:
		return []any{true, false}, nil
	case core.TypeString, core.TypeCategorical:
		if len(param.Options) == 0 {
			return nil, fmt.Errorf("parameter %s of type %s must have options", param.Name, param.Type)
		}
		return param.Options, nil
	default:
		return nil, fmt.Errorf("unsupported parameter type: %s", param.Type)
	}
}

// generateIntValues creates integer values within the specified range and step
func (g *GridSearch) generateIntValues(param core.Parameter) ([]any, error) {
	min, ok := param.Min.(int)
	if !ok {
		return nil, fmt.Errorf("parameter %s min value must be an integer", param.Name)
	}

	max, ok := param.Max.(int)
	if !ok {
		return nil, fmt.Errorf("parameter %s max value must be an integer", param.Name)
	}

	step, ok := param.Step.(int)
	if !ok {
		return nil, fmt.Errorf("parameter %s step value must be an integer", param.Name)
	}

	if step <= 0 {
		return nil, fmt.Errorf("parameter %s step value must be positive", param.Name)
	}

	var values []any
	for i := min; i <= max; i += step {
		values = append(values, i)
	}

	return values, nil
}

// generateFloatValues creates float values within the specified range and step
func (g *GridSearch) generateFloatValues(param core.Parameter) ([]any, error) {
	min, ok := param.Min.(float64)
	if !ok {
		return nil, fmt.Errorf("parameter %s min value must be a float", param.Name)
	}

	max, ok := param.Max.(float64)
	if !ok {
		return nil, fmt.Errorf("parameter %s max value must be a float", param.Name)
	}

	step, ok := param.Step.(float64)
	if !ok {
		return nil, fmt.Errorf("parameter %s step value must be a float", param.Name)
	}

	if step <= 0 {
		return nil, fmt.Errorf("parameter %s step value must be positive", param.Name)
	}

	var values []any
	for f := min; f <= max; f += step {
		values = append(values, f)
	}

	return values, nil
}

// runEvaluations executes the evaluations for all parameter sets
func (g *GridSearch) runEvaluations(ctx context.Context, evaluator core.Evaluator, parameterSets []core.ParameterSet) ([]*core.OptimizerResult, error) {
	var (
		results   []*core.OptimizerResult
		mutex     sync.Mutex
		wg        sync.WaitGroup
		errCh     = make(chan error, 1)
		semaphore = make(chan struct{}, g.parallelism)
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
		go func(index int, paramSet core.ParameterSet) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore

			g.logf("Evaluating parameter set %d/%d", index+1, len(parameterSets))

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

			g.logf("Completed evaluation %d/%d", index+1, len(parameterSets))
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

// logf logs a message if a log is configured
func (g *GridSearch) logf(format string, args ...any) {
	if g.log != nil {
		g.log.Infof(format, args...)
	}
}
