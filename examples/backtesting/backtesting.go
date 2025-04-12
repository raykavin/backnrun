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

// This example shows how to use backtesting with NinjaBot
// Backtesting is a simulation of the strategy in historical data (from CSV)
func main() {
	ctx := context.Background()

	// bot settings (eg: pairs, telegram, etc)
	settings := &core.Settings{
		Pairs: []string{
			"BTCUSDT",
			"ETHUSDT",
		},
	}

	// initialize your strategy
	strategy := new(strategies.CrossEMA)

	// load historical data from CSV files
	csvFeed, err := exchange.NewCSVFeed(
		strategy.Timeframe(),
		exchange.PairFeed{
			Pair:      "BTCUSDT",
			File:      "testdata/btc-1h.csv",
			Timeframe: "1h",
		},
		exchange.PairFeed{
			Pair:      "ETHUSDT",
			File:      "testdata/eth-1h.csv",
			Timeframe: "1h",
		},
	)
	if err != nil {
		backnrun.DefaultLog.Fatal(err)
	}

	// initialize a database in memory
	storage, err := storage.FromMemory()
	if err != nil {
		backnrun.DefaultLog.Fatal(err)
	}

	// create a paper wallet for simulation, initializing with 10.000 USDT
	wallet := exchange.NewPaperWallet(
		ctx,
		"USDT",
		backnrun.DefaultLog,
		exchange.WithPaperAsset("USDT", 10000),
		exchange.WithDataFeed(csvFeed),
	)

	// create a chart  with indicators from the strategy and a custom additional RSI indicator
	chart, err := plot.NewChart(
		backnrun.DefaultLog,
		plot.WithStrategyIndicators(strategy),
		plot.WithCustomIndicators(
			indicator.RSI(14, "purple"),
		),
		plot.WithPaperWallet(wallet),
	)
	if err != nil {
		backnrun.DefaultLog.Fatal(err)
		return
	}

	// initializer Ninjabot with the objects created before
	bot, err := backnrun.NewBot(
		ctx,
		settings,
		wallet,
		strategy,
		backnrun.WithBacktest(wallet), // Required for Backtest mode
		backnrun.WithStorage(storage),

		// connect bot feed (candle and orders) to the chart
		backnrun.WithCandleSubscription(chart),
		backnrun.WithOrderSubscription(chart),
		backnrun.WithLogLevel(logger.WarnLevel),
	)
	if err != nil {
		backnrun.DefaultLog.Fatal(err)
	}

	// Initializer simulation
	err = bot.Run(ctx)
	if err != nil {
		backnrun.DefaultLog.Fatal(err)
	}

	// Print bot results
	bot.Summary()

	// Display candlesticks chart in local browser
	err = chart.Start()
	if err != nil {
		backnrun.DefaultLog.Fatal(err)
	}
}
