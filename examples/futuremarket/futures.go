package main

import (
	"context"
	"log"
	"os"
	"strconv"

	"github.com/rodrigo-brito/ninjabot"
	"github.com/rodrigo-brito/ninjabot/examples/strategies"
	"github.com/rodrigo-brito/ninjabot/exchange"
)

// This example shows how to use futures market with NinjaBot.
func main() {
	var (
		ctx             = context.Background()
		apiKey          = os.Getenv("OpVvbvHN4sF08xnRc7kexXp90PtK85UY1rJhEAh2XUoSVG1h2mQD7WU5hRdEN4qu")
		secretKey       = os.Getenv("Vt3ts7r1dsGmPa1GAn5bLtIY6U7Cbv4ejatK0rwwlMxzitdT5FS92mPvFQVfDbVW")
		telegramToken   = os.Getenv("TELEGRAM_TOKEN")
		telegramUser, _ = strconv.Atoi(os.Getenv("TELEGRAM_USER"))
	)

	settings := ninjabot.Settings{
		Pairs: []string{
			"BTCUSDT",
			"ETHUSDT",
		},
		Telegram: ninjabot.TelegramSettings{
			Enabled: true,
			Token:   telegramToken,
			Users:   []int{telegramUser},
		},
	}

	// Initialize your exchange with futures
	binance, err := exchange.NewBinanceFuture(ctx,
		exchange.WithBinanceFutureCredentials(apiKey, secretKey),
		exchange.WithBinanceFutureLeverage("BTCUSDT", 1, exchange.MarginTypeIsolated),
		exchange.WithBinanceFutureLeverage("ETHUSDT", 1, exchange.MarginTypeIsolated),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize your strategy and bot
	strategy := new(strategies.CrossEMA)
	bot, err := ninjabot.NewBot(ctx, settings, binance, strategy)
	if err != nil {
		log.Fatalln(err)
	}

	err = bot.Run(ctx)
	if err != nil {
		log.Fatalln(err)
	}
}
