// Package main is the entry point for the BackNRun trading application
package main

import (
	"context"

	"github.com/raykavin/backnrun/bot"
	"github.com/raykavin/backnrun/core"
	"github.com/raykavin/backnrun/examples/trend_master/internal/chart"
	"github.com/raykavin/backnrun/examples/trend_master/internal/config"
	"github.com/raykavin/backnrun/examples/trend_master/internal/exchange"
	"github.com/raykavin/backnrun/examples/trend_master/internal/strategy"
	"github.com/raykavin/backnrun/examples/trend_master/pkg/utils"
	"github.com/raykavin/backnrun/storage"
)

var (
	log = utils.GetLogger()
)

func main() {
	ctx := context.Background()

	// Load application configuration
	appConfig, err := config.LoadAppConfig()
	checkError(err)

	// Load TrendMaster strategy configuration
	strategyConfig, err := config.LoadStrategyConfig(appConfig.ConfigPath)
	checkError(err)

	// Set up exchange
	ex, err := exchange.SetupBinanceExchange(ctx, appConfig.Binance)
	checkError(err)

	// Set up chart server with configuration
	chartServer, chart, err := chart.SetupChartServer(strategyConfig.Indicators)
	checkError(err)

	// Set up trading bot with configured strategy
	tradingBot, err := setupBot(ctx, appConfig, ex, chart, strategyConfig)
	checkError(err)

	// Start chart server in a separate goroutine
	go func() {
		if err := chartServer.Start(); err != nil {
			log.Fatal(err)
		}
	}()

	// Log bot initialization
	log.WithFields(map[string]any{
		"time_frame":          strategyConfig.General.Timeframe,
		"pairs":               strategyConfig.General.Pairs,
		"configPath":          appConfig.ConfigPath,
		"tradingHoursEnabled": strategyConfig.General.TradingHours.Enabled,
	}).Info("TrendMaster initialized with loaded configuration")

	// Start the trading bot
	if err := tradingBot.Run(ctx); err != nil {
		log.Fatal(err)
	}
}

// setupBot creates and configures the trading bot
func setupBot(
	ctx context.Context,
	cfg *config.AppConfig,
	ex core.Exchange,
	chart *chart.Chart,
	strategyConfig *strategy.TrendMasterConfig,
) (*bot.Bot, error) {
	// Configure bot settings
	settings := &core.Settings{
		Pairs: strategyConfig.General.Pairs,
		Telegram: core.TelegramSettings{
			Enabled: cfg.Telegram.Enabled,
			Token:   cfg.Telegram.Token,
			Users:   []int{cfg.Telegram.UserID},
		},
	}

	// Initialize storage
	storageDB, err := storage.FromSQLite(cfg.StoragePath)
	if err != nil {
		return nil, err
	}

	// Create strategy with configuration
	trendMasterStrategy := strategy.NewTrendMasterStrategy(strategyConfig)
	// trendMasterStrategy := strategies.NewCrossEMA(5, 10, 10)

	// Create and configure the bot
	return bot.NewBot(
		ctx,
		settings,
		ex,
		log,
		trendMasterStrategy,
		bot.WithCandleSubscription(chart),
		bot.WithOrderSubscription(chart),
		bot.WithStorage(storageDB),
	)
}

// checkError handles fatal errors
func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
