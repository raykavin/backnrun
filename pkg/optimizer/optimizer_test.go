package optimizer

import (
	"context"
	"testing"
	"time"
)

// MockEvaluator is a simple evaluator for testing
type MockEvaluator struct {
	// Map of parameter sets to results for deterministic testing
	ResultMap map[string]map[string]float64
}

// Evaluate implements the Evaluator interface
func (m *MockEvaluator) Evaluate(ctx context.Context, params ParameterSet) (*Result, error) {
	// Format the parameter set to use as a key
	key := FormatParameterSet(params)
	
	// Get the predefined metrics for this parameter set
	metrics, exists := m.ResultMap[key]
	if !exists {
		// If no predefined result, create a simple one based on parameter values
		metrics = make(map[string]float64)
		
		// Example: For EMA strategy, higher emaLength and lower smaLength might be better
		if emaLength, ok := params["emaLength"].(int); ok {
			metrics["profit"] = float64(emaLength) * 10
		}
		
		if smaLength, ok := params["smaLength"].(int); ok {
			metrics["profit"] -= float64(smaLength) * 5
		}
		
		metrics["win_rate"] = 0.5 // Default win rate
	}
	
	return &Result{
		Parameters: params,
		Metrics:    metrics,
		Duration:   100 * time.Millisecond,
	}, nil
}

// TestGridSearch tests the grid search optimizer
func TestGridSearch(t *testing.T) {
	// Create mock evaluator
	evaluator := &MockEvaluator{
		ResultMap: map[string]map[string]float64{
			"{emaLength: 9, smaLength: 21}": {
				"profit":   100.0,
				"win_rate": 0.6,
			},
			"{emaLength: 14, smaLength: 28}": {
				"profit":   150.0,
				"win_rate": 0.7,
			},
		},
	}
	
	// Define parameters
	parameters := []Parameter{
		{
			Name:        "emaLength",
			Description: "EMA Length",
			Default:     9,
			Min:         9,
			Max:         14,
			Step:        5,
			Type:        TypeInt,
		},
		{
			Name:        "smaLength",
			Description: "SMA Length",
			Default:     21,
			Min:         21,
			Max:         28,
			Step:        7,
			Type:        TypeInt,
		},
	}
	
	// Create config
	config := NewConfig().
		WithParameters(parameters...).
		WithMaxIterations(10).
		WithParallelism(2)
	
	// Create grid search optimizer
	gridSearch, err := NewGridSearch(config)
	if err != nil {
		t.Fatalf("Failed to create grid search: %v", err)
	}
	
	// Run optimization
	results, err := gridSearch.Optimize(
		context.Background(),
		evaluator,
		MetricProfit,
		true,
	)
	if err != nil {
		t.Fatalf("Grid search optimization failed: %v", err)
	}
	
	// Check results
	if len(results) != 4 {
		t.Errorf("Expected 4 results, got %d", len(results))
	}
	
	// Check if the best result is as expected
	if len(results) > 0 {
		bestResult := results[0]
		expectedProfit := 150.0
		
		if bestResult.Metrics["profit"] != expectedProfit {
			t.Errorf("Expected best profit to be %.2f, got %.2f", 
				expectedProfit, bestResult.Metrics["profit"])
		}
	}
}

// TestRandomSearch tests the random search optimizer
func TestRandomSearch(t *testing.T) {
	// Create mock evaluator
	evaluator := &MockEvaluator{
		ResultMap: map[string]map[string]float64{},
	}
	
	// Define parameters
	parameters := []Parameter{
		{
			Name:        "emaLength",
			Description: "EMA Length",
			Default:     9,
			Min:         5,
			Max:         20,
			Step:        1,
			Type:        TypeInt,
		},
		{
			Name:        "smaLength",
			Description: "SMA Length",
			Default:     21,
			Min:         15,
			Max:         40,
			Step:        1,
			Type:        TypeInt,
		},
	}
	
	// Create config
	config := NewConfig().
		WithParameters(parameters...).
		WithMaxIterations(5).
		WithParallelism(2)
	
	// Create random search optimizer
	randomSearch, err := NewRandomSearch(config)
	if err != nil {
		t.Fatalf("Failed to create random search: %v", err)
	}
	
	// Run optimization
	results, err := randomSearch.Optimize(
		context.Background(),
		evaluator,
		MetricProfit,
		true,
	)
	if err != nil {
		t.Fatalf("Random search optimization failed: %v", err)
	}
	
	// Check results
	if len(results) != 5 {
		t.Errorf("Expected 5 results, got %d", len(results))
	}
}

// TestParameterValidation tests parameter validation
func TestParameterValidation(t *testing.T) {
	// Define parameters
	parameters := []Parameter{
		{
			Name:        "intParam",
			Description: "Integer Parameter",
			Default:     10,
			Min:         1,
			Max:         100,
			Step:        1,
			Type:        TypeInt,
		},
		{
			Name:        "floatParam",
			Description: "Float Parameter",
			Default:     0.5,
			Min:         0.1,
			Max:         1.0,
			Step:        0.1,
			Type:        TypeFloat,
		},
	}
	
	// Valid parameter set
	validParams := ParameterSet{
		"intParam":   50,
		"floatParam": 0.5,
	}
	
	// Invalid parameter set (missing parameter)
	missingParams := ParameterSet{
		"intParam": 50,
	}
	
	// Invalid parameter set (wrong type)
	wrongTypeParams := ParameterSet{
		"intParam":   50.5, // Should be int
		"floatParam": 0.5,
	}
	
	// Test valid parameters
	err := ValidateParameterSet(validParams, parameters)
	if err != nil {
		t.Errorf("Valid parameter set failed validation: %v", err)
	}
	
	// Test missing parameter
	err = ValidateParameterSet(missingParams, parameters)
	if err == nil {
		t.Errorf("Missing parameter should fail validation")
	}
	
	// Test wrong type parameter
	err = ValidateParameterSet(wrongTypeParams, parameters)
	if err == nil {
		t.Errorf("Wrong type parameter should fail validation")
	}
}

// TestResultSorter tests the result sorting functionality
func TestResultSorter(t *testing.T) {
	// Create test results
	results := []*Result{
		{
			Parameters: ParameterSet{"param": 1},
			Metrics: map[string]float64{
				"profit": 100.0,
				"risk":   0.5,
			},
		},
		{
			Parameters: ParameterSet{"param": 2},
			Metrics: map[string]float64{
				"profit": 200.0,
				"risk":   0.8,
			},
		},
		{
			Parameters: ParameterSet{"param": 3},
			Metrics: map[string]float64{
				"profit": 150.0,
				"risk":   0.3,
			},
		},
	}
	
	// Test sorting by profit (maximize)
	profitSorter := ResultSorter{
		Results:    results,
		MetricName: "profit",
		Maximize:   true,
	}
	
	if !profitSorter.Less(1, 0) {
		t.Errorf("Result with higher profit should be sorted first when maximizing")
	}
	
	if !profitSorter.Less(1, 2) {
		t.Errorf("Result with higher profit should be sorted first when maximizing")
	}
	
	// Test sorting by risk (minimize)
	riskSorter := ResultSorter{
		Results:    results,
		MetricName: "risk",
		Maximize:   false,
	}
	
	if !riskSorter.Less(2, 0) {
		t.Errorf("Result with lower risk should be sorted first when minimizing")
	}
	
	if !riskSorter.Less(2, 1) {
		t.Errorf("Result with lower risk should be sorted first when minimizing")
	}
}
