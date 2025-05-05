// Package exchange provides integration with cryptocurrency exchanges
package exchange

import (
	"context"

	"github.com/raykavin/backnrun/bot"
	"github.com/raykavin/backnrun/core"
	"github.com/raykavin/backnrun/examples/trend_master/internal/config"
	"github.com/raykavin/backnrun/exchange/binance"
)

// SetupBinanceExchange initializes and configures the Binance exchange
func SetupBinanceExchange(ctx context.Context, cfg config.BinanceConfig) (core.Exchange, error) {
	// Create Binance exchange connection
	ex, err := binance.NewExchange(ctx, bot.DefaultLog, binance.Config{
		Type:       binance.MarketTypeFutures,
		APIKey:     cfg.APIKey,
		APISecret:  cfg.SecretKey,
		UseTestnet: cfg.UseTestnet,
		Debug:      cfg.Debug,
	})
	if err != nil {
		return nil, err
	}

	// Configure leverage for futures trading
	ConfigureLeverage(ex.(*binance.Futures))
	return ex, nil
}

// ConfigureLeverage sets the leverage for each trading pair
func ConfigureLeverage(ex *binance.Futures) {
	// Default trading pairs
	pairs := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT"}

	// Configure leverage for each pair
	for _, p := range pairs {
		binance.WithFuturesLeverage(p, 20, binance.MarginTypeIsolated)(ex)
	}
}
