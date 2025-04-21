// Package main provides an example of using BackNRun with Binance futures market
package main

import (
	"context"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/raykavin/backnrun/bot"
	"github.com/raykavin/backnrun/core"
	"github.com/raykavin/backnrun/exchange/binance"
	"github.com/raykavin/backnrun/plot"
	"github.com/raykavin/backnrun/plot/indicator"
	"github.com/raykavin/backnrun/storage"
	"github.com/raykavin/backnrun/strategies"
)

// Config holds the application configuration
type Config struct {
	BinanceAPIKey    string
	BinanceSecretKey string
	TelegramToken    string
	TelegramUserID   int
	UseTestnet       bool
	TelegramEnabled  bool
}

// loadConfig loads configuration from environment variables
func loadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, err
	}

	telegramUserStr := os.Getenv("TELEGRAM_USER")
	telegramUser := 0

	var err error
	if telegramUserStr != "" {
		telegramUser, err = strconv.Atoi(telegramUserStr)
		if err != nil {
			return nil, err
		}
	}

	return &Config{
		BinanceAPIKey:    os.Getenv("BINANCE_API_KEY"),
		BinanceSecretKey: os.Getenv("BINANCE_SECRET_KEY"),
		TelegramToken:    os.Getenv("TELEGRAM_TOKEN"),
		TelegramUserID:   telegramUser,
		UseTestnet:       true,
		TelegramEnabled:  false,
	}, nil
}

// setupExchange initializes and configures the Binance futures exchange
func setupExchange(ctx context.Context, config *Config) (core.Exchange, error) {
	binanceEx, err := binance.NewExchange(ctx, bot.DefaultLog, binance.Config{
		Type:          binance.MarketTypeFutures,
		APIKey:        config.BinanceAPIKey,
		APISecret:     config.BinanceSecretKey,
		UseTestnet:    config.UseTestnet,
		UseHeikinAshi: false,
	})

	if err != nil {
		return nil, err
	}

	// Configure leverage for trading pairs
	futuresExchange := binanceEx.(*binance.Futures)
	configureFuturesLeverage(futuresExchange)

	return binanceEx, nil
}

// configureFuturesLeverage sets up leverage for the trading pairs
func configureFuturesLeverage(futuresExchange *binance.Futures) {
	// Set leverage to 20x with isolated margin for each trading pair
	binance.WithFuturesLeverage("OMUSDT", 20, binance.MarginTypeIsolated)(futuresExchange)
	binance.WithFuturesLeverage("BELUSDT", 20, binance.MarginTypeIsolated)(futuresExchange)
	binance.WithFuturesLeverage("VOXELUSDT", 20, binance.MarginTypeIsolated)(futuresExchange)
}

// setupChart initializes the charting system
func setupChartServer() (*plot.ChartServer, *plot.Chart, error) {
	chart, err := plot.NewChart(bot.DefaultLog,
		plot.WithCustomIndicators(
			indicator.EMA(32, "lime", indicator.High),
			indicator.EMA(32, "red", indicator.Low),
			indicator.MACD(14, 150, 14, "blue", "red", "green"),
		),
	)

	if err != nil {
		return nil, nil, err
	}

	return plot.NewChartServer(chart, plot.NewStandardHTTPServer(), bot.DefaultLog), chart, nil
}

// setupBot creates and configures the trading bot
func setupBot(ctx context.Context, config *Config, exchange core.Exchange, chart *plot.Chart, chartServer *plot.ChartServer) (*bot.Bot, error) {
	// Define trading pairs
	settings := &core.Settings{
		Pairs: []string{
			"OMUSDT",
			"BELUSDT",
			"VOXELUSDT",
		},
		Telegram: core.TelegramSettings{
			Enabled: config.TelegramEnabled,
			Token:   config.TelegramToken,
			Users:   []int{config.TelegramUserID},
		},
	}

	// Initialize trading strategy with parameters
	strategy := strategies.NewTrendMasterStrategy()

	storage, err := storage.FromSQLite("./backnrun.sqlite")
	if err != nil {
		panic(err)
	}

	// Create and configure the bot
	return bot.NewBot(
		ctx,
		settings,
		exchange,
		strategy,
		bot.DefaultLog,
		bot.WithCandleSubscription(chart),
		bot.WithOrderSubscription(chart),
		bot.WithStorage(storage),
	)
}

// This example shows how to use futures market withbot.
func main() {
	ctx := context.Background()

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		bot.DefaultLog.Fatal(err)
	}

	// Initialize exchange
	exchange, err := setupExchange(ctx, config)
	if err != nil {
		bot.DefaultLog.Fatal(err)
	}

	// Setup chart
	chartServer, chart, err := setupChartServer()
	if err != nil {
		bot.DefaultLog.Fatal(err)
	}

	// Initialize bot
	b, err := setupBot(ctx, config, exchange, chart, chartServer)
	if err != nil {
		bot.DefaultLog.Fatal(err)
	}

	go func() {
		err := chartServer.Start()
		if err != nil {
			log.Fatal(err)
		}
	}()

	// Run the bot
	err = b.Run(ctx)
	if err != nil {
		log.Fatalln(err)
	}
}
