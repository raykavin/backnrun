package main

import (
	"context"
	"log"
	"os"
	"strconv"

	"github.com/raykavin/backnrun"
	"github.com/raykavin/backnrun/examples/strategies"
	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/exchange/binance"
)

// This example shows how to use futures market with BackNRun.
func main() {
	var (
		ctx             = context.Background()
		apiKey          = os.Getenv("OpVvbvHN4sF08xnRc7kexXp90PtK85UY1rJhEAh2XUoSVG1h2mQD7WU5hRdEN4qu")
		secretKey       = os.Getenv("Vt3ts7r1dsGmPa1GAn5bLtIY6U7Cbv4ejatK0rwwlMxzitdT5FS92mPvFQVfDbVW")
		telegramToken   = os.Getenv("TELEGRAM_TOKEN")
		telegramUser, _ = strconv.Atoi(os.Getenv("TELEGRAM_USER"))
	)

	settings := &core.Settings{
		Pairs: []string{
			"BTCUSDT",
			"ETHUSDT",
		},
		Telegram: core.TelegramSettings{
			Enabled: true,
			Token:   telegramToken,
			Users:   []int{telegramUser},
		},
	}

	// Initialize your exchange with futures
	binanceEx, err := binance.NewExchange(ctx, backnrun.DefaultLog, binance.Config{
		Type:          binance.MarketTypeFutures,
		APIKey:        apiKey,
		APISecret:     secretKey,
		UseTestnet:    true,
		UseHeikinAshi: false,
	})

	binance.WithFuturesLeverage("BTCUSDT", 1, binance.MarginTypeIsolated)(binanceEx.(*binance.Futures))
	binance.WithFuturesLeverage("ETHUSDT", 1, binance.MarginTypeIsolated)(binanceEx.(*binance.Futures))

	if err != nil {
		log.Fatal(err)
	}

	// Initialize your strategy and bot
	strategy := new(strategies.CrossEMA)
	bot, err := backnrun.NewBot(ctx, settings, binanceEx, strategy, backnrun.DefaultLog)
	if err != nil {
		log.Fatalln(err)
	}

	err = bot.Run(ctx)
	if err != nil {
		log.Fatalln(err)
	}
}
