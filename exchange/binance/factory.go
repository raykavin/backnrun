// Package binance provides interfaces to interact with Binance exchange
package binance

import (
	"context"
	"fmt"

	"github.com/raykavin/backnrun/core"
)

// ---------------------
// Exchange Factory
// ---------------------

// NewExchange creates and returns a BinanceExchangeType based on the market type
func NewExchange(ctx context.Context, log core.Logger, config Config) (BinanceExchangeType, error) {
	switch config.Type {
	case MarketTypeSpot:
		return createSpotExchange(ctx, config)
	case MarketTypeFutures:
		return createFuturesExchange(ctx, config)
	default:
		return nil, fmt.Errorf("unsupported market type: %s", config.Type)
	}
}

// ---------------------
// Exchange Creation Functions
// ---------------------

// createSpotExchange initializes a spot exchange client with the given configuration
func createSpotExchange(ctx context.Context, config Config) (BinanceExchangeType, error) {
	options := buildSpotOptions(config)
	return NewSpot(ctx, options...)
}

// createFuturesExchange initializes a futures exchange client with the given configuration
func createFuturesExchange(ctx context.Context, config Config) (BinanceExchangeType, error) {
	options := buildFuturesOptions(config)
	return NewFutures(ctx, options...)
}

// ---------------------
// Option Builders
// ---------------------

// buildSpotOptions constructs option list for spot exchange configuration
func buildSpotOptions(config Config) []SpotOption {
	options := []SpotOption{}

	// Add credentials if both key and secret are provided
	if hasValidCredentials(config.APIKey, config.APISecret) {
		options = append(options, WithSpotCredentials(config.APIKey, config.APISecret))
	}

	// Add feature options
	if config.UseHeikinAshi {
		options = append(options, WithSpotHeikinAshiCandles())
	}

	if config.UseTestnet {
		options = append(options, WithSpotTestNet())
	}

	// Configure custom endpoints when provided
	addSpotCustomEndpoints(&options, config)

	// Add metadata fetchers
	addMetadataFetchers(&options, config.MetadataFetchers, WithSpotMetadataFetcher)

	return options
}

// buildFuturesOptions constructs option list for futures exchange configuration
func buildFuturesOptions(config Config) []FuturesOption {
	options := []FuturesOption{}

	// Add credentials if both key and secret are provided
	if hasValidCredentials(config.APIKey, config.APISecret) {
		options = append(options, WithFuturesCredentials(config.APIKey, config.APISecret))
	}

	// Add feature options
	if config.UseHeikinAshi {
		options = append(options, WithFuturesHeikinAshiCandles())
	}

	if config.UseTestnet {
		options = append(options, WithFuturesTestNet())
	}

	if config.Debug {
		options = append(options, WithFuturesClientDebug())
	}

	// Add pair-specific options like leverage and margin type
	for _, pairOption := range config.FuturesPairOptions {
		options = append(options, WithFuturesLeverage(
			pairOption.Pair,
			pairOption.Leverage,
			pairOption.MarginType,
		))
	}

	// Add metadata fetchers
	addMetadataFetchers(&options, config.MetadataFetchers, WithFuturesMetadataFetcher)

	return options
}

// ---------------------
// Helper Functions
// ---------------------

// hasValidCredentials checks if both API key and secret are non-empty
func hasValidCredentials(key, secret string) bool {
	return key != "" && secret != ""
}

// addSpotCustomEndpoints adds custom endpoint options when configured
func addSpotCustomEndpoints(options *[]SpotOption, config Config) {
	if config.CustomMainAPI.API != "" {
		*options = append(*options, WithSpotCustomMainAPIEndpoint(
			config.CustomMainAPI.API,
			config.CustomMainAPI.WebSocket,
			config.CustomMainAPI.Combined,
		))
	}

	if config.CustomTestnetAPI.API != "" {
		*options = append(*options, WithSpotCustomTestnetAPIEndpoint(
			config.CustomTestnetAPI.API,
			config.CustomTestnetAPI.WebSocket,
			config.CustomTestnetAPI.Combined,
		))
	}
}

// addMetadataFetchers adds metadata fetchers to the options
// The fetcher function parameter uses a type parameter to handle both spot and futures fetchers
func addMetadataFetchers[T any](options *[]T, fetchers []MetadataFetcher,
	fetcherFunc func(MetadataFetcher) T) {
	for _, fetcher := range fetchers {
		*options = append(*options, fetcherFunc(fetcher))
	}
}
