package binance

import (
	"fmt"
	"strconv"
	"time"

	"github.com/raykavin/backnrun/pkg/core"

	"github.com/adshao/go-binance/v2/common"
	"github.com/jpillora/backoff"
)

// Common errors
var (
	ErrInvalidAsset    = fmt.Errorf("invalid asset")
	ErrInvalidQuantity = fmt.Errorf("invalid quantity")
)

// BinanceExchangeType defines the common interface for all Binance exchange types
type BinanceExchangeType interface {
	// Market data
	core.Feeder
	// LastQuote(ctx context.Context, pair string) (float64, error)
	// AssetsInfo(pair string) core.AssetInfo
	// Candles
	// CandlesByLimit(ctx context.Context, pair, period string, limit int) ([]core.Candle, error)
	// CandlesByPeriod(ctx context.Context, pair, period string, start, end time.Time) ([]core.Candle, error)
	// CandlesSubscription(ctx context.Context, pair, period string) (chan core.Candle, chan error)
	
	// Order management
	core.Broker
	// Account() (core.Account, error)
	// Position(pair string) (asset float64, quote float64, err error)
	// CreateOrderLimit(side core.SideType, pair string, quantity, limit float64) (core.Order, error)
	// CreateOrderMarket(side core.SideType, pair string, quantity float64) (core.Order, error)
	// CreateOrderMarketQuote(side core.SideType, pair string, quantity float64) (core.Order, error)
	// CreateOrderStop(pair string, quantity, limit float64) (core.Order, error)
	// CreateOrderOCO(side core.SideType, pair string, quantity, price, stop, stopLimit float64) ([]core.Order, error)
	// Orders(pair string, limit int) ([]core.Order, error)
	// Order(pair string, id int64) (core.Order, error)
	// Cancel(order core.Order) error
}

// OrderError represents an error that occurred during order creation or execution
type OrderError struct {
	Err      error
	Pair     string
	Quantity float64
}

func (e *OrderError) Error() string {
	return fmt.Sprintf("order error: %v, pair: %s, quantity: %f", e.Err, e.Pair, e.Quantity)
}

// MetadataFetcher is a function type for fetching additional candle metadata
type MetadataFetcher func(pair string, t time.Time) (string, float64)

// SplitAssetQuote splits a trading pair into asset and quote parts
func SplitAssetQuote(pair string) (asset, quote string) {
	var quoteAssets = []string{"USDT", "BUSD", "USDC", "BTC", "ETH", "BNB"}

	for _, quote = range quoteAssets {
		if len(pair) > len(quote) && pair[len(pair)-len(quote):] == quote {
			asset = pair[:len(pair)-len(quote)]
			return
		}
	}

	// Default fallback: assume BTC as quote if no other quote asset found
	if len(pair) > 3 {
		return pair[:len(pair)-3], pair[len(pair)-3:]
	}

	return pair, ""
}

// formatQuantity standardizes the quantity based on asset precision
func formatQuantity(assetsInfo map[string]core.AssetInfo, pair string, value float64) string {
	if info, ok := assetsInfo[pair]; ok {
		value = common.AmountToLotSize(info.StepSize, info.BaseAssetPrecision, value)
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}

// formatPrice standardizes the price based on asset precision
func formatPrice(assetsInfo map[string]core.AssetInfo, pair string, value float64) string {
	if info, ok := assetsInfo[pair]; ok {
		value = common.AmountToLotSize(info.TickSize, info.QuotePrecision, value)
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}

// validateOrder checks if the quantity is valid for the given pair
func validateOrder(assetsInfo map[string]core.AssetInfo, pair string, quantity float64) error {
	info, ok := assetsInfo[pair]
	if !ok {
		return ErrInvalidAsset
	}

	if quantity > info.MaxQuantity || quantity < info.MinQuantity {
		return &OrderError{
			Err:      fmt.Errorf("%w: min: %f max: %f", ErrInvalidQuantity, info.MinQuantity, info.MaxQuantity),
			Pair:     pair,
			Quantity: quantity,
		}
	}

	return nil
}

// setupBackoffRetry creates a backoff with sensible defaults
func setupBackoffRetry() *backoff.Backoff {
	return &backoff.Backoff{
		Min: 100 * time.Millisecond,
		Max: 1 * time.Second,
	}
}
