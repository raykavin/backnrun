package main

import (
	"context"
	"log"
	"os"
	"strconv"

	"github.com/raykavin/backnrun/bot"
	"github.com/raykavin/backnrun/core"
	"github.com/raykavin/backnrun/exchange/binance"
	"github.com/raykavin/backnrun/strategies"
)

// This example shows how to use spot market with BackNRun in Binance
func main() {
	var (
		ctx             = context.Background()
		apiKey          = os.Getenv("API_KEY")
		secretKey       = os.Getenv("API_SECRET")
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

	// Initialize your binance
	binance, err := binance.NewExchange(ctx, bot.DefaultLog, binance.Config{
		Type:          binance.MarketTypeSpot,
		APIKey:        apiKey,
		APISecret:     secretKey,
		UseTestnet:    false,
		UseHeikinAshi: false,
	})

	if err != nil {
		log.Fatalln(err)
	}

	// Initialize your strategy and bot
	strategy := new(strategies.CrossEMA)
	bot, err := bot.NewBot(
		ctx,
		settings,
		binance,
		bot.DefaultLog,
		strategy,
	)
	if err != nil {
		log.Fatalln(err)
	}

	err = bot.Run(ctx)
	if err != nil {
		log.Fatalln(err)
	}
}
