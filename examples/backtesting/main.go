package main

import (
	"context"
	"fmt"
	"log"

	"github.com/raykavin/backnrun/bot"
	"github.com/raykavin/backnrun/core"
	"github.com/raykavin/backnrun/exchange"
	"github.com/raykavin/backnrun/plot"
	"github.com/raykavin/backnrun/storage"
	"github.com/raykavin/backnrun/strategies"
)

var pairs = []string{"BTCUSDT", "ETHUSDT", "OMUSDT", "BELUSDT", "VOXELUSDT"}

func main() {
	ctx := context.Background()

	// Logger initialization
	logger := bot.DefaultLog
	logger.SetLevel(core.DebugLevel)

	// Strategy
	strategy := strategies.NewTripleMAStrategy()
	// strategy := strategies.NewAdaptiveMomentumStrategy()
	// strategy := strategies.NewTurtleStrategy()

	// Services
	dataFeed := mustInitializeFeed(strategy.Timeframe(), logger)
	db := must(storage.FromMemory())
	wallet := newWallet(ctx, logger, dataFeed)
	chartServer, chart := mustInitializeChartServer(logger, strategy, wallet)

	settings := &core.Settings{
		Pairs: pairs,
	}

	// Initialize the bot
	tradingBot := must(initializeBot(
		ctx,
		settings,
		wallet,
		strategy,
		db,
		chart,
		logger,
	))

	// Run simulation
	mustRun(tradingBot.Run(ctx))
	tradingBot.Summary()

	// Start the chart server
	mustRun(chartServer.Start())
}

// Utility function to handle errors
func must[T any](val T, err error) T {
	if err != nil {
		log.Fatal(err)
	}
	return val
}

func mustRun(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// Historical data feed
func mustInitializeFeed(timeframe string, logger core.Logger) *exchange.CSVFeed {
	pairsFeed := make([]exchange.PairFeed, len(pairs))

	for i, pair := range pairs {
		pairsFeed[i] = exchange.PairFeed{
			Pair:      pair,
			File:      fmt.Sprintf("./data/%s-%s.csv", pair, timeframe),
			Timeframe: timeframe,
		}
	}

	return must(exchange.NewCSVFeed(timeframe, pairsFeed...))
}

// Create a simulation wallet
func newWallet(ctx context.Context, logger core.Logger, feed *exchange.CSVFeed) *exchange.PaperWallet {
	return exchange.NewPaperWallet(ctx, "USDT", logger,
		exchange.WithPaperAsset("USDT", 100), exchange.WithDataFeed(feed))
}

// Initialize chart server
func mustInitializeChartServer(
	logger core.Logger,
	strategy core.Strategy,
	wallet *exchange.PaperWallet,
) (*plot.ChartServer, *plot.Chart) {
	chart := must(plot.NewChart(logger, plot.WithStrategyIndicators(strategy), plot.WithPaperWallet(wallet)))
	server := plot.NewChartServer(chart, plot.NewStandardHTTPServer(), logger)
	return server, chart
}

// Initialize the bot
func initializeBot(
	ctx context.Context,
	settings *core.Settings,
	wallet *exchange.PaperWallet,
	strategy core.Strategy,
	db core.Storage,
	chart *plot.Chart,
	logger core.Logger,
) (*bot.Bot, error) {
	return bot.NewBot(
		ctx,
		settings,
		wallet,
		logger,
		strategy,
		bot.WithBacktest(wallet),
		bot.WithStorage(db),
		bot.WithCandleSubscription(chart),
		bot.WithOrderSubscription(chart),
	)
}
