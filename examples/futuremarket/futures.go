// Package main provides an example of using BackNRun with Binance futures market
package main

import (
	"context"
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

// ---------------------
// Constants
// ---------------------

// Environment variable names
const (
	BINANCE_API_KEY      = "BINANCE_API_KEY"
	BINANCE_SECRET_KEY   = "BINANCE_SECRET_KEY"
	BINANCE_DEBUG_CLIENT = "BINANCE_DEBUG_CLIENT"
	BINANCE_USE_TESTNET  = "BINANCE_USE_TESTNET"

	TELEGRAM_TOKEN   = "TELEGRAM_TOKEN"
	TELEGRAM_USER    = "TELEGRAM_USER"
	TELEGRAM_ENABLED = "TELEGRAM_ENABLED"
)

// Global variables
var (
	pairs = []string{"BTCUSDT", "ETHUSDT", "BNBUSDT"}
	log   = bot.DefaultLog
)

// ---------------------
// Configuration
// ---------------------

// Config holds the application configuration loaded from environment variables
type Config struct {
	BinanceAPIKey    string
	BinanceSecretKey string
	TelegramToken    string
	UseTestnet       bool
	Debug            bool
	TelegramEnabled  bool
	TelegramUserID   int
}

// loadConfig loads configuration from environment variables
func loadConfig() (*Config, error) {
	// Load environment variables from .env file if present
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	config := &Config{
		BinanceAPIKey:    os.Getenv(BINANCE_API_KEY),
		BinanceSecretKey: os.Getenv(BINANCE_SECRET_KEY),
	}

	// Process boolean environment variables
	boolVars := map[string]*bool{
		BINANCE_USE_TESTNET:  &config.UseTestnet,
		BINANCE_DEBUG_CLIENT: &config.Debug,
		TELEGRAM_ENABLED:     &config.TelegramEnabled,
	}

	for envVar, configVar := range boolVars {
		value, err := strconv.ParseBool(os.Getenv(envVar))
		checkError(log, err)

		*configVar = value
	}

	// Process Telegram configuration if enabled
	if config.TelegramEnabled {
		config.TelegramToken = os.Getenv(TELEGRAM_TOKEN)

		telegramUserID, err := strconv.Atoi(os.Getenv(TELEGRAM_USER))
		checkError(log, err)

		config.TelegramUserID = telegramUserID
	}

	return config, nil
}

// checkError handles fatal errors
func checkError(log core.Logger, err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// ---------------------
// Exchange Setup
// ---------------------

// setupExchange initializes and configures the Binance exchange
func setupExchange(ctx context.Context, cfg *Config) (core.Exchange, error) {
	ex, err := binance.NewExchange(ctx, bot.DefaultLog, binance.Config{
		Type:       binance.MarketTypeFutures,
		APIKey:     cfg.BinanceAPIKey,
		APISecret:  cfg.BinanceSecretKey,
		UseTestnet: cfg.UseTestnet,
		Debug:      cfg.Debug,
	})
	if err != nil {
		return nil, err
	}

	configureLeverage(ex.(*binance.Futures))
	return ex, nil
}

// configureLeverage sets the leverage for each trading pair
func configureLeverage(ex *binance.Futures) {
	for _, p := range pairs {
		binance.WithFuturesLeverage(p, 20, binance.MarginTypeIsolated)(ex)
	}
}

// ---------------------
// Chart Setup
// ---------------------

// setupChartServer creates and configures the chart visualization server
func setupChartServer() (*plot.ChartServer, *plot.Chart, error) {
	chart, err := plot.NewChart(
		bot.DefaultLog,
		plot.WithCustomIndicators(
			indicator.EMA(9, "lime", indicator.Close),
			indicator.EMA(21, "red", indicator.Close),
			indicator.MACD(14, 150, 14, "blue", "red", "green"),
		),
	)
	if err != nil {
		return nil, nil, err
	}

	return plot.NewChartServer(chart, plot.NewStandardHTTPServer(), bot.DefaultLog), chart, nil
}

// ---------------------
// Bot Setup
// ---------------------

// setupBot creates and configures the trading bot
func setupBot(
	ctx context.Context,
	cfg *Config,
	ex core.Exchange,
	chart *plot.Chart,
) (*bot.Bot, error) {
	// Configure bot settings
	settings := &core.Settings{
		Pairs: pairs,
		Telegram: core.TelegramSettings{
			Enabled: cfg.TelegramEnabled,
			Token:   cfg.TelegramToken,
			Users:   []int{cfg.TelegramUserID},
		},
	}

	// Initialize storage
	storageDB, err := storage.FromSQLite("./backnrun.sqlite")
	if err != nil {
		return nil, err
	}

	// Create and configure the bot
	return bot.NewBot(
		ctx,
		settings,
		ex,
		bot.DefaultLog,
		strategies.NewTrendMasterStrategy(),
		bot.WithCandleSubscription(chart),
		bot.WithOrderSubscription(chart),
		bot.WithStorage(storageDB),
	)
}

// ---------------------
// Main Function
// ---------------------

func main() {
	ctx := context.Background()

	// Load configuration
	config, err := loadConfig()
	checkError(log, err)

	// Set up exchange
	ex, err := setupExchange(ctx, config)
	checkError(log, err)

	// Set up chart server
	chartServer, chart, err := setupChartServer()
	checkError(log, err)

	// Set up trading bot
	tradingBot, err := setupBot(ctx, config, ex, chart)
	checkError(log, err)

	// Start chart server in a separate goroutine
	go func() {
		if err := chartServer.Start(); err != nil {
			log.Fatal(err)
		}
	}()

	// Start the trading bot
	if err := tradingBot.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
