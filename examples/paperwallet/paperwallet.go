package main

import (
	"context"
	"log"
	"os"
	"strconv"

	"github.com/raykavin/backnrun"
	"github.com/raykavin/backnrun/examples/strategies"
	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/exchange"
	"github.com/raykavin/backnrun/pkg/exchange/binance"
	"github.com/raykavin/backnrun/pkg/plot"
	"github.com/raykavin/backnrun/pkg/plot/indicator"
	"github.com/raykavin/backnrun/pkg/storage"
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
			"BTCUSDT",
			// "ETHUSDT",
			// "BNBUSDT",
			// "LTCUSDT",
		},
		Telegram: core.TelegramSettings{
			Enabled: telegramToken != "" && telegramUser != 0,
			Token:   telegramToken,
			Users:   []int{telegramUser},
		},
	}

	// Use binance for realtime data feed
	binanceEx, err := binance.NewExchange(ctx, backnrun.DefaultLog, binance.Config{Type: binance.MarketTypeSpot})
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
		backnrun.DefaultLog,
		exchange.WithPaperFee(0.001, 0.001),
		exchange.WithPaperAsset("USDT", 10000),
		exchange.WithDataFeed(binanceEx),
	)

	// initializing my strategy
	strategy := &strategies.CrossEMA{}

	chart, err := plot.NewChart(
		backnrun.DefaultLog,
		plot.WithCustomIndicators(
			indicator.EMA(8, "red"),
			indicator.SMA(21, "blue"),
		),
	)

	if err != nil {
		log.Fatal(err)
	}

	// initializer core
	bot, err := backnrun.NewBot(
		ctx,
		settings,
		paperWallet,
		strategy,
		backnrun.WithStorage(storage),
		backnrun.WithPaperWallet(paperWallet),
		backnrun.WithCandleSubscription(chart),
		backnrun.WithOrderSubscription(chart),
	)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		err := chart.Start()
		if err != nil {
			log.Fatal(err)
		}
	}()

	err = bot.Run(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
