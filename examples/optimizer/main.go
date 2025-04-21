package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/raykavin/backnrun/bot"
	"github.com/raykavin/backnrun/core"
	"github.com/raykavin/backnrun/exchange"
	"github.com/raykavin/backnrun/strategies"

	"github.com/raykavin/backnrun/optimizer"
)

// CreateMACDStrategyFactory creates a factory function for MACD strategies
// This uses the strategies package from examples
func CreateMACDStrategyFactory() optimizer.StrategyFactory {
	return func(params core.ParameterSet) (core.Strategy, error) {
		// Create a new MACD strategy with default parameters
		strategy := strategies.NewMACDDivergenceStrategy()

		// Since the strategy methods for setting parameters might not be exported,
		// we would need to modify the strategy implementation to support parameter changes
		// For now, we'll return the strategy with default parameters

		// Note: In a real implementation, you would need to ensure the strategy
		// has proper setters for these parameters or modify the strategy to accept
		// parameters in its constructor

		return strategy, nil
	}
}

// GetEMAStrategyParameters returns the parameters that can be optimized for EMA cross strategy
func GetEMAStrategyParameters() []core.Parameter {
	return []core.Parameter{
		{
			Name:        "emaLength",
			Description: "Length of the EMA indicator",
			Default:     9,
			Min:         3,
			Max:         50,
			Step:        1,
			Type:        core.TypeInt,
		},
		{
			Name:        "smaLength",
			Description: "Length of the SMA indicator",
			Default:     21,
			Min:         5,
			Max:         100,
			Step:        1,
			Type:        core.TypeInt,
		},
		{
			Name:        "minQuoteAmount",
			Description: "Minimum quote currency amount for trades",
			Default:     10.0,
			Min:         1.0,
			Max:         100.0,
			Step:        5.0,
			Type:        core.TypeFloat,
		},
	}
}

// GetMACDStrategyParameters returns the parameters that can be optimized for MACD strategy
func GetMACDStrategyParameters() []core.Parameter {
	return []core.Parameter{
		{
			Name:        "fastPeriod",
			Description: "Fast period for MACD calculation",
			Default:     12,
			Min:         5,
			Max:         30,
			Step:        1,
			Type:        core.TypeInt,
		},
		{
			Name:        "slowPeriod",
			Description: "Slow period for MACD calculation",
			Default:     26,
			Min:         10,
			Max:         50,
			Step:        2,
			Type:        core.TypeInt,
		},
		{
			Name:        "signalPeriod",
			Description: "Signal period for MACD calculation",
			Default:     9,
			Min:         3,
			Max:         20,
			Step:        1,
			Type:        core.TypeInt,
		},
		{
			Name:        "lookbackPeriod",
			Description: "Lookback period for divergence detection",
			Default:     14,
			Min:         5,
			Max:         30,
			Step:        1,
			Type:        core.TypeInt,
		},
		{
			Name:        "positionSize",
			Description: "Position size as a fraction of available capital",
			Default:     0.5,
			Min:         0.1,
			Max:         1.0,
			Step:        0.1,
			Type:        core.TypeFloat,
		},
		{
			Name:        "stopLoss",
			Description: "Stop loss percentage",
			Default:     0.03,
			Min:         0.01,
			Max:         0.1,
			Step:        0.01,
			Type:        core.TypeFloat,
		},
		{
			Name:        "takeProfit",
			Description: "Take profit percentage",
			Default:     0.06,
			Min:         0.02,
			Max:         0.2,
			Step:        0.01,
			Type:        core.TypeFloat,
		},
	}
}

func main() {
	// Set up context and logging
	ctx := context.Background()
	log := bot.DefaultLog
	log.SetLevel(core.InfoLevel)

	// Initialize data feed for backtesting
	dataFeed, err := initializeDataFeed("15m")
	if err != nil {
		log.Fatal(err)
	}

	// Configure trading pairs
	settings := &core.Settings{
		Pairs: []string{"BTCUSDT"},
	}

	// Create strategy factory for EMA Cross strategy
	strategyFactory := createEMAStrategyFactory()

	// Create evaluator
	evaluator := optimizer.NewBacktestStrategyEvaluator(
		strategyFactory,
		settings,
		dataFeed,
		log,
		1000.0, // Starting balance
		"USDT", // Quote currency
	)

	// Configure optimizer
	config := optimizer.NewConfig().
		WithParameters(GetEMAStrategyParameters()...).
		WithMaxIterations(50).
		WithParallelism(4).
		WithLogger(log).
		WithTargetMetric(core.MetricProfit, true).
		WithTopN(5)

	// Create grid search optimizer
	gridSearch, err := optimizer.NewGridSearch(config)
	if err != nil {
		log.Fatal(err)
	}

	// Run optimization
	fmt.Println("Starting grid search optimization...")
	startTime := time.Now()

	results, err := gridSearch.Optimize(
		ctx,
		evaluator,
		core.MetricProfit,
		true,
	)
	if err != nil {
		log.Fatal(err)
	}

	duration := time.Since(startTime)
	fmt.Printf("Optimization completed in %s\n", duration.Round(time.Second))

	// Print results
	optimizer.PrintResults(results, core.MetricProfit, 5)

	// Save results to CSV
	outputFile := "ema_optimization_results.csv"
	if err := optimizer.SaveResultsToCSV(results, core.MetricProfit, outputFile); err != nil {
		log.Errorf("Failed to save results: %v", err)
	} else {
		fmt.Printf("Results saved to %s\n", outputFile)
	}

	// Run random search for comparison
	fmt.Println("\nStarting random search optimization for comparison...")
	randomConfig := *config
	randomConfig.MaxIterations = 20

	randomSearch, err := optimizer.NewRandomSearch(&randomConfig)
	if err != nil {
		log.Fatal(err)
	}

	startTime = time.Now()
	randomResults, err := randomSearch.Optimize(
		ctx,
		evaluator,
		core.MetricProfit,
		true,
	)
	if err != nil {
		log.Fatal(err)
	}

	duration = time.Since(startTime)
	fmt.Printf("Random search completed in %s\n", duration.Round(time.Second))

	// Print random search results
	optimizer.PrintResults(randomResults, core.MetricProfit, 5)
}

// initializeDataFeed sets up the historical data source from CSV files
func initializeDataFeed(timeframe string) (*exchange.CSVFeed, error) {
	// Check if data file exists
	dataFile := "btc-15m.csv"
	if _, err := os.Stat(dataFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("data file %s not found, please download historical data first", dataFile)
	}

	return exchange.NewCSVFeed(
		timeframe,
		exchange.PairFeed{
			Pair:      "BTCUSDT",
			File:      dataFile,
			Timeframe: "15m",
		},
	)
}

// createEMAStrategyFactory creates a factory function for EMA cross strategies
func createEMAStrategyFactory() optimizer.StrategyFactory {
	return func(params core.ParameterSet) (core.Strategy, error) {
		// Extract parameters with validation
		emaLength, ok := params["emaLength"].(int)
		if !ok {
			return nil, fmt.Errorf("emaLength must be an integer")
		}

		smaLength, ok := params["smaLength"].(int)
		if !ok {
			return nil, fmt.Errorf("smaLength must be an integer")
		}

		minQuoteAmount, ok := params["minQuoteAmount"].(float64)
		if !ok {
			return nil, fmt.Errorf("minQuoteAmount must be a float")
		}

		// Create and return the strategy
		return strategies.NewCrossEMA(emaLength, smaLength, minQuoteAmount), nil
	}
}

// createMACDStrategyFactory creates a factory function for MACD strategies
func createMACDStrategyFactory() optimizer.StrategyFactory {
	return func(params core.ParameterSet) (core.Strategy, error) {
		// Create a new MACD strategy with default parameters
		return strategies.NewMACDDivergenceStrategy(), nil
	}
}
