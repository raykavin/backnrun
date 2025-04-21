package main

import (
	"context"
	"log"
	"os"
	"strconv"

	"github.com/raykavin/backnrun/bot"
	"github.com/raykavin/backnrun/core"
	"github.com/raykavin/backnrun/exchange"
	"github.com/raykavin/backnrun/exchange/binance"
	"github.com/raykavin/backnrun/plot"
	"github.com/raykavin/backnrun/plot/indicator"
	"github.com/raykavin/backnrun/storage"
	"github.com/raykavin/backnrun/strategies"
)

// This example shows how to use BackNRun with a simulation with a fake exchange
// A peperwallet is a wallet that is not connected to any exchange, it is a simulation with live data (realtime)
func main() {
	var (
		ctx             = context.Background()
		telegramToken   = os.Getenv("TELEGRAM_TOKEN")
		telegramUser, _ = strconv.Atoi(os.Getenv("TELEGRAM_USER"))
	)

	settings := &core.Settings{
		Pairs: []string{
			"OMUSDT",
			"BELUSDT",
		},
		Telegram: core.TelegramSettings{
			Enabled: telegramToken != "" && telegramUser != 0,
			Token:   telegramToken,
			Users:   []int{telegramUser},
		},
	}

	// Use binance for realtime data feed
	binanceEx, err := binance.NewExchange(ctx, bot.DefaultLog, binance.Config{Type: binance.MarketTypeSpot})
	if err != nil {
		log.Fatal(err)
	}

	// creating a storage to save trades
	storage, err := storage.FromMemory()
	if err != nil {
		log.Fatal(err)
	}

	// creating a paper wallet to simulate an exchange waller for fake operataions
	// paper wallet is simulation of a real exchange wallet
	paperWallet := exchange.NewPaperWallet(
		ctx,
		"USDT",
		bot.DefaultLog,
		exchange.WithPaperFee(0.001, 0.001),
		exchange.WithPaperAsset("USDT", 10000),
		exchange.WithDataFeed(binanceEx),
	)

	// initializing my strategy
	strategy := strategies.NewCrossEMA(9, 21, 10.0)

	chart, err := plot.NewChart(
		bot.DefaultLog,
		plot.WithCustomIndicators(
			indicator.EMA(8, "red", indicator.Close),
			indicator.SMA(21, "blue", indicator.Close),
		),
	)

	chartServer := plot.NewChartServer(chart, plot.NewStandardHTTPServer(), bot.DefaultLog)

	if err != nil {
		log.Fatal(err)
	}

	// initializer core
	bot, err := bot.NewBot(
		ctx,
		settings,
		paperWallet,
		strategy,
		bot.DefaultLog,
		bot.WithStorage(storage),
		bot.WithPaperWallet(paperWallet),
		bot.WithCandleSubscription(chart),
		bot.WithOrderSubscription(chart),
	)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		err := chartServer.Start()
		if err != nil {
			log.Fatal(err)
		}
	}()

	err = bot.Run(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
