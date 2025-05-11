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

func main() {
	ctx := context.Background()
	logger := bot.DefaultLog

	settings := getBotSettings()

	ex, err := createExchange(ctx, logger)
	if err != nil {
		log.Fatal(err)
	}

	memStorage, err := storage.FromMemory()
	if err != nil {
		log.Fatal(err)
	}

	paperWallet := createPaperWallet(ctx, ex, logger)

	strategy := strategies.NewCrossEMA(9, 21, 10.0)

	chart, err := createChart(logger)
	if err != nil {
		log.Fatal(err)
	}

	chartServer := plot.NewChartServer(chart, plot.NewStandardHTTPServer(), logger)

	b, err := bot.NewBot(
		ctx,
		settings,
		paperWallet,
		logger,
		strategy,
		bot.WithStorage(memStorage),
		bot.WithPaperWallet(paperWallet),
		bot.WithCandleSubscription(chart),
		bot.WithOrderSubscription(chart),
	)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		if err := chartServer.Start(); err != nil {
			log.Fatal(err)
		}
	}()

	if err := b.Run(ctx); err != nil {
		log.Fatal(err)
	}
}

// --- Helpers ---

func getBotSettings() *core.Settings {
	telegramToken := os.Getenv("TELEGRAM_TOKEN")
	telegramUser, _ := strconv.Atoi(os.Getenv("TELEGRAM_USER"))

	return &core.Settings{
		Pairs: []string{"OMUSDT", "BELUSDT"},
		Telegram: core.TelegramSettings{
			Enabled: telegramToken != "" && telegramUser != 0,
			Token:   telegramToken,
			Users:   []int{telegramUser},
		},
	}
}

func createExchange(ctx context.Context, logger core.Logger) (binance.BinanceExchangeType, error) {
	return binance.NewExchange(ctx, logger, binance.Config{
		Type: binance.MarketTypeSpot,
	})
}

func createPaperWallet(ctx context.Context, dataFeed core.Feeder, logger core.Logger) *exchange.PaperWallet {
	return exchange.NewPaperWallet(
		ctx,
		"USDT",
		logger,
		exchange.WithPaperFee(0.001, 0.001),
		exchange.WithPaperAsset("USDT", 10000),
		exchange.WithDataFeed(dataFeed),
	)
}

func createChart(logger core.Logger) (*plot.Chart, error) {
	return plot.NewChart(
		logger,
		plot.WithDebug(),
		plot.WithCustomIndicators(
			indicator.EMA(8, "red", indicator.Close),
			indicator.SMA(21, "blue", indicator.Close),
		),
	)
}
