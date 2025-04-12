package binance

import (
	"context"
	"fmt"

	"github.com/raykavin/backnrun/pkg/logger"
)

// MarketType represents the type of market (spot or futures)
type MarketType string

const (
	// MarketTypeSpot represents the spot market
	MarketTypeSpot MarketType = "spot"

	// MarketTypeFutures represents the futures market
	MarketTypeFutures MarketType = "futures"
)

// Config represents the common configuration for Binance clients
type Config struct {
	// Market type (spot or futures)
	Type MarketType

	// API credentials
	APIKey    string
	APISecret string

	// Use testnet
	UseTestnet bool

	// Use Heikin Ashi candles
	UseHeikinAshi bool

	// Custom endpoints (if needed)
	CustomMainAPI    CustomEndpoint
	CustomTestnetAPI CustomEndpoint

	// Future-specific configuration
	FuturesPairOptions []PairOption

	// Common metadata fetchers
	MetadataFetchers []MetadataFetcher
}

// CustomEndpoint represents custom API endpoints
type CustomEndpoint struct {
	API       string
	WebSocket string
	Combined  string
}

// NewExchange creates a new exchange client based on the provided configuration
func NewExchange(ctx context.Context, logger logger.Logger, config Config) (BinanceExchangeType, error) {
	switch config.Type {
	case MarketTypeSpot:
		return newSpotExchange(ctx, logger, config)
	case MarketTypeFutures:
		return newFuturesExchange(ctx, logger, config)
	default:
		return nil, fmt.Errorf("unknown market type: %s", config.Type)
	}
}

// newSpotExchange creates a new spot exchange client
func newSpotExchange(ctx context.Context, logger logger.Logger, config Config) (BinanceExchangeType, error) {
	options := []SpotOption{}

	// Add credentials if provided
	if config.APIKey != "" && config.APISecret != "" {
		options = append(options, WithSpotCredentials(config.APIKey, config.APISecret))
	}

	// Configure Heikin Ashi if requested
	if config.UseHeikinAshi {
		options = append(options, WithSpotHeikinAshiCandles())
	}

	// Configure testnet if requested
	if config.UseTestnet {
		options = append(options, WithSpotTestNet())
	}

	// Configure custom endpoints if provided
	if config.CustomMainAPI.API != "" {
		options = append(options, WithSpotCustomMainAPIEndpoint(
			config.CustomMainAPI.API,
			config.CustomMainAPI.WebSocket,
			config.CustomMainAPI.Combined,
		))
	}

	if config.CustomTestnetAPI.API != "" {
		options = append(options, WithSpotCustomTestnetAPIEndpoint(
			config.CustomTestnetAPI.API,
			config.CustomTestnetAPI.WebSocket,
			config.CustomTestnetAPI.Combined,
		))
	}

	// Add metadata fetchers
	for _, fetcher := range config.MetadataFetchers {
		options = append(options, WithSpotMetadataFetcher(fetcher))
	}

	// Create and return the spot client
	return NewSpot(ctx, logger, options...)
}

// newFuturesExchange creates a new futures exchange client
func newFuturesExchange(ctx context.Context, logger logger.Logger, config Config) (BinanceExchangeType, error) {
	options := []FuturesOption{}

	// Add credentials if provided
	if config.APIKey != "" && config.APISecret != "" {
		options = append(options, WithFuturesCredentials(config.APIKey, config.APISecret))
	}

	// Configure Heikin Ashi if requested
	if config.UseHeikinAshi {
		options = append(options, WithFuturesHeikinAshiCandles())
	}

	// Add pair options (leverage and margin type)
	for _, pairOption := range config.FuturesPairOptions {
		options = append(options, WithFuturesLeverage(
			pairOption.Pair,
			pairOption.Leverage,
			pairOption.MarginType,
		))
	}

	// Add metadata fetchers
	for _, fetcher := range config.MetadataFetchers {
		options = append(options, WithFuturesMetadataFetcher(fetcher))
	}

	// Create and return the futures client
	return NewFutures(ctx, logger, options...)
}
