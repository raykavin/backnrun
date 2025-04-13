package main

import (
	"context"

	"github.com/raykavin/backnrun"
	"github.com/raykavin/backnrun/examples/strategies"
	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/exchange"
	"github.com/raykavin/backnrun/pkg/logger"
	"github.com/raykavin/backnrun/pkg/plot"
	"github.com/raykavin/backnrun/pkg/plot/indicator"
	"github.com/raykavin/backnrun/pkg/storage"
)

// main demonstrates how to use BackNRun for backtesting a trading strategy
// against historical data loaded from CSV files.
func main() {
	// Set up context and logging
	ctx := context.Background()
	log := backnrun.DefaultLog
	log.SetLevel(logger.DebugLevel)

	// Initialize trading strategy
	strategy := strategies.NewWilliams91Strategy()

	// Configure trading pairs
	settings := &core.Settings{
		Pairs: []string{"BTCUSDT"},
	}

	// Initialize services
	dataFeed, err := initializeDataFeed(strategy.Timeframe())
	if err != nil {
		log.Fatal(err)
	}

	db, err := storage.FromMemory()
	if err != nil {
		log.Fatal(err)
	}

	wallet := initializeWallet(ctx, log, dataFeed)

	chart, err := initializeChart(log, strategy, wallet)
	if err != nil {
		log.Fatal(err)
	}

	// Set up the trading bot
	bot, err := initializeBot(ctx, settings, wallet, strategy, db, chart, log)
	if err != nil {
		log.Fatal(err)
	}

	// Run simulation
	if err := bot.Run(ctx); err != nil {
		log.Fatal(err)
	}

	// Display results
	bot.Summary()

	// Show interactive chart
	if err := chart.Start(); err != nil {
		log.Fatal(err)
	}
}

// initializeDataFeed sets up the historical data source from CSV files
func initializeDataFeed(timeframe string) (*exchange.CSVFeed, error) {
	return exchange.NewCSVFeed(
		timeframe,
		exchange.PairFeed{
			Pair:      "BTCUSDT",
			File:      "btc-5m.csv",
			Timeframe: "5m",
		},
		// Additional pairs can be added like this:
		// exchange.PairFeed{
		//     Pair:      "ETHUSDT",
		//     File:      "testdata/eth-1h.csv",
		//     Timeframe: "1h",
		// },
	)
}

// initializeWallet creates a paper trading wallet with initial funds
func initializeWallet(ctx context.Context, log logger.Logger, feed *exchange.CSVFeed) *exchange.PaperWallet {
	return exchange.NewPaperWallet(
		ctx,
		"USDT",
		log,
		exchange.WithPaperAsset("USDT", 1000),
		exchange.WithDataFeed(feed),
	)
}

// initializeChart sets up visualization with strategy indicators
func initializeChart(log logger.Logger, strategy core.Strategy, wallet *exchange.PaperWallet) (*plot.Chart, error) {
	return plot.NewChart(
		log,
		plot.WithStrategyIndicators(strategy),
		plot.WithCustomIndicators(
			indicator.RSI(14, "purple"),
		),
		plot.WithPaperWallet(wallet),
	)
}

// initializeBot sets up the BackNRun trading bot with all required components
func initializeBot(
	ctx context.Context,
	settings *core.Settings,
	wallet *exchange.PaperWallet,
	strategy core.Strategy,
	db core.OrderStorage,
	chart *plot.Chart,
	log logger.Logger,
) (*backnrun.Backnrun, error) {
	return backnrun.NewBot(
		ctx,
		settings,
		wallet,
		strategy,
		log,
		backnrun.WithBacktest(wallet), // Required for Backtest mode
		backnrun.WithStorage(db),
		backnrun.WithCandleSubscription(chart),
		backnrun.WithOrderSubscription(chart),
	)
}
