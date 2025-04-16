package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/raykavin/backnrun"
	"github.com/raykavin/backnrun/examples/strategies"
	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/exchange"
	"github.com/raykavin/backnrun/pkg/logger"
	"github.com/raykavin/backnrun/pkg/optimizer"
)

func main() {
	// Set up context and logging
	ctx := context.Background()
	log := backnrun.DefaultLog
	log.SetLevel(logger.InfoLevel)

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
		WithParameters(optimizer.GetEMAStrategyParameters()...).
		WithMaxIterations(50).
		WithParallelism(4).
		WithLogger(log).
		WithTargetMetric(optimizer.MetricProfit, true).
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
		optimizer.MetricProfit,
		true,
	)
	if err != nil {
		log.Fatal(err)
	}

	duration := time.Since(startTime)
	fmt.Printf("Optimization completed in %s\n", duration.Round(time.Second))

	// Print results
	optimizer.PrintResults(results, optimizer.MetricProfit, 5)

	// Save results to CSV
	outputFile := "ema_optimization_results.csv"
	if err := optimizer.SaveResultsToCSV(results, optimizer.MetricProfit, outputFile); err != nil {
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
		optimizer.MetricProfit,
		true,
	)
	if err != nil {
		log.Fatal(err)
	}

	duration = time.Since(startTime)
	fmt.Printf("Random search completed in %s\n", duration.Round(time.Second))

	// Print random search results
	optimizer.PrintResults(randomResults, optimizer.MetricProfit, 5)
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
	return func(params optimizer.ParameterSet) (core.Strategy, error) {
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
	return func(params optimizer.ParameterSet) (core.Strategy, error) {
		// Create a new MACD strategy with default parameters
		return strategies.NewMACDDivergenceStrategy(), nil
	}
}
