package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/raykavin/backnrun/bot"
	"github.com/raykavin/backnrun/core"
	"github.com/raykavin/backnrun/exchange"
	"github.com/raykavin/backnrun/optimizer"
	"github.com/raykavin/backnrun/strategies"
)

func main() {
	ctx := context.Background()
	log := bot.DefaultLog
	log.SetLevel(core.InfoLevel)

	dataFeed, err := loadDataFeed("15m", "btc-15m.csv", "BTCUSDT")
	if err != nil {
		log.Fatal(err)
	}

	settings := &core.Settings{Pairs: []string{"BTCUSDT"}}
	strategyFactory := createEMAStrategyFactory()

	evaluator := optimizer.NewBacktestStrategyEvaluator(
		strategyFactory,
		settings,
		dataFeed,
		log,
		1000.0,
		"USDT",
	)

	config := optimizer.NewConfig().
		WithParameters(getEMAStrategyParameters()...).
		WithMaxIterations(50).
		WithParallelism(4).
		WithLogger(log).
		WithTargetMetric(core.MetricProfit, true).
		WithTopN(5)

	fmt.Println("Starting grid search optimization...")
	gridResults := runOptimization(ctx, config, evaluator, log, "grid")

	saveResults("ema_optimization_results.csv", gridResults, log)

	fmt.Println("\nStarting random search optimization for comparison...")
	randomConfig := *config
	randomConfig.MaxIterations = 20
	runOptimization(ctx, &randomConfig, evaluator, log, "random")
}

// --- Strategy Factories ---

func createEMAStrategyFactory() optimizer.StrategyFactory {
	return func(params core.ParameterSet) (core.Strategy, error) {
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
		return strategies.NewCrossEMA(emaLength, smaLength, minQuoteAmount), nil
	}
}

func createMACDStrategyFactory() optimizer.StrategyFactory {
	return func(params core.ParameterSet) (core.Strategy, error) {
		return strategies.NewMACDDivergenceStrategy(), nil
	}
}

// --- Optimization Helpers ---

func runOptimization(
	ctx context.Context,
	config *optimizer.Config,
	evaluator core.Evaluator,
	log core.Logger,
	label string,
) []*core.OptimizerResult {
	start := time.Now()

	var (
		opt  core.Optimizer
		err  error
		name = label + " search"
	)

	if label == "grid" {
		opt, err = optimizer.NewGridSearch(config)
	} else {
		opt, err = optimizer.NewRandomSearch(config)
	}

	if err != nil {
		log.Fatal(err)
	}

	results, err := opt.Optimize(ctx, evaluator, core.MetricProfit, true)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s completed in %s\n", name, time.Since(start).Round(time.Second))
	optimizer.PrintResults(results, core.MetricProfit, 5)

	return results
}

func saveResults(filename string, results []*core.OptimizerResult, log core.Logger) {
	if err := optimizer.SaveResultsToCSV(results, core.MetricProfit, filename); err != nil {
		log.Errorf("Failed to save results: %v", err)
	} else {
		fmt.Printf("Results saved to %s\n", filename)
	}
}

// --- Data Feed ---

func loadDataFeed(timeframe, filename, pair string) (*exchange.CSVFeed, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, fmt.Errorf("data file %s not found", filename)
	}
	return exchange.NewCSVFeed(timeframe, exchange.PairFeed{
		Pair:      pair,
		File:      filename,
		Timeframe: timeframe,
	})
}

// --- Parameters ---

func getEMAStrategyParameters() []core.Parameter {
	return []core.Parameter{
		{Name: "emaLength", Description: "Length of the EMA", Default: 9, Min: 3, Max: 50, Step: 1, Type: core.TypeInt},
		{Name: "smaLength", Description: "Length of the SMA", Default: 21, Min: 5, Max: 100, Step: 1, Type: core.TypeInt},
		{Name: "minQuoteAmount", Description: "Minimum trade amount", Default: 10.0, Min: 1.0, Max: 100.0, Step: 5.0, Type: core.TypeFloat},
	}
}

func getMACDStrategyParameters() []core.Parameter {
	return []core.Parameter{
		{Name: "fastPeriod", Description: "MACD Fast Period", Default: 12, Min: 5, Max: 30, Step: 1, Type: core.TypeInt},
		{Name: "slowPeriod", Description: "MACD Slow Period", Default: 26, Min: 10, Max: 50, Step: 2, Type: core.TypeInt},
		{Name: "signalPeriod", Description: "MACD Signal Period", Default: 9, Min: 3, Max: 20, Step: 1, Type: core.TypeInt},
		{Name: "lookbackPeriod", Description: "Divergence Lookback", Default: 14, Min: 5, Max: 30, Step: 1, Type: core.TypeInt},
		{Name: "positionSize", Description: "Capital Fraction", Default: 0.5, Min: 0.1, Max: 1.0, Step: 0.1, Type: core.TypeFloat},
		{Name: "stopLoss", Description: "Stop Loss %", Default: 0.03, Min: 0.01, Max: 0.1, Step: 0.01, Type: core.TypeFloat},
		{Name: "takeProfit", Description: "Take Profit %", Default: 0.06, Min: 0.02, Max: 0.2, Step: 0.01, Type: core.TypeFloat},
	}
}
