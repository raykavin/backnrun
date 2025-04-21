package main

import (
	"context"

	"github.com/raykavin/backnrun/bot"
	"github.com/raykavin/backnrun/core"
	"github.com/raykavin/backnrun/exchange"
	"github.com/raykavin/backnrun/strategies"

	"github.com/raykavin/backnrun/plot"
	"github.com/raykavin/backnrun/storage"
)

// main demonstrates how to use BackNRun for backtesting a trading strategy
// against historical data loaded from CSV files.
func main() {

	// Set up context and logging
	ctx := context.Background()
	log := bot.DefaultLog
	log.SetLevel(core.DebugLevel)

	// Initialize trading strategy
	strategy := strategies.NewAdaptiveMomentumStrategy()

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

	chartServer, chart, err := initializeChartServer(log, strategy, wallet)
	if err != nil {
		log.Fatal(err)
	}

	// Set up the trading bot
	bot, err := initializeBot(
		ctx,
		settings,
		wallet,
		strategy,
		db,
		chart,
		chartServer,
		log,
	)

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
	if err := chartServer.Start(); err != nil {
		log.Fatal(err)
	}
}

// initializeDataFeed sets up the historical data source from CSV files
func initializeDataFeed(timeframe string) (*exchange.CSVFeed, error) {
	return exchange.NewCSVFeed(
		timeframe,
		exchange.PairFeed{
			Pair:      "BTCUSDT",
			File:      "btc-5m-2.csv",
			Timeframe: "5m",
		},
		// exchange.PairFeed{
		// 	Pair:      "ETHUSDT",
		// 	File:      "eth-15m.csv",
		// 	Timeframe: "15m",
		// },
	)
}

// initializeWallet creates a paper trading wallet with initial funds
func initializeWallet(ctx context.Context, log core.Logger, feed *exchange.CSVFeed) *exchange.PaperWallet {
	return exchange.NewPaperWallet(
		ctx,
		"USDT",
		log,
		exchange.WithPaperAsset("USDT", 100),
		exchange.WithDataFeed(feed),
	)
}

// initializeChartServer sets up visualization with strategy indicators
func initializeChartServer(log core.Logger, strategy core.Strategy, wallet *exchange.PaperWallet) (*plot.ChartServer, *plot.Chart, error) {
	chart, err := plot.NewChart(
		log,
		plot.WithStrategyIndicators(strategy),
		plot.WithPaperWallet(wallet),
	)

	if err != nil {
		return nil, nil, err
	}

	return plot.NewChartServer(chart, plot.NewStandardHTTPServer(), log), chart, nil
}

// initializeBot sets up the BackNRun trading bot with all required components
func initializeBot(
	ctx context.Context,
	settings *core.Settings,
	wallet *exchange.PaperWallet,
	strategy core.Strategy,
	db core.Storage,
	chart *plot.Chart,
	chartServer *plot.ChartServer,
	log core.Logger,
) (*bot.Bot, error) {
	return bot.NewBot(
		ctx,
		settings,
		wallet,
		strategy,
		log,
		bot.WithBacktest(wallet), // Required for Backtest mode
		bot.WithStorage(db),
		bot.WithCandleSubscription(chart),
		bot.WithOrderSubscription(chart),
	)
}
