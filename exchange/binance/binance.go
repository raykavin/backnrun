// Package binance provides interfaces to interact with Binance exchange
package binance

import (
	"fmt"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
	"github.com/raykavin/backnrun/core"

	"github.com/jpillora/backoff"
)

// ---------------------
// Constants and Errors
// ---------------------

// Common errors
var (
	ErrInvalidAsset    = fmt.Errorf("invalid asset")
	ErrInvalidQuantity = fmt.Errorf("invalid quantity")
)

// Known quote currencies for pair splitting
var pairs = []string{
	"USDT",
	"BTC",
	"BNB",
	"ETH",
	"BUSD",
	"USDC",
	"EUR",
	"TRY",
	"AUD",
	"BRL",
	"GBP",
	"USD",
	"NGN",
}

// ---------------------
// Types
// ---------------------

// BinanceExchangeType defines the common interface for all Binance exchange types
type BinanceExchangeType interface {
	core.Feeder
	core.Broker
}

// OrderError represents an error that occurred during order creation or execution
type OrderError struct {
	Err      error
	Pair     string
	Quantity float64
}

// Error implements the error interface for OrderError
func (e *OrderError) Error() string {
	return fmt.Sprintf("order error: %v, pair: %s, quantity: %f", e.Err, e.Pair, e.Quantity)
}

// MetadataFetcher is a function type for fetching additional candle metadata
type MetadataFetcher func(pair string, t time.Time) (string, float64)

// MarketType defines the trading market category
type MarketType string

const (
	// MarketTypeSpot represents the spot trading market
	MarketTypeSpot MarketType = "spot"

	// MarketTypeFutures represents the futures trading market
	MarketTypeFutures MarketType = "futures"
)

// Config holds configuration parameters for Binance clients
type Config struct {
	Type               MarketType
	APIKey             string
	APISecret          string
	UseTestnet         bool
	UseHeikinAshi      bool
	Debug              bool
	CustomMainAPI      Endpoint
	CustomTestnetAPI   Endpoint
	FuturesPairOptions []PairOption
	MetadataFetchers   []MetadataFetcher
}

// Endpoint represents API endpoint URLs
type Endpoint struct {
	API       string
	WebSocket string
	Combined  string
}

// ---------------------
// Pair Handling
// ---------------------

// SplitAssetQuote splits a trading pair into base asset and quote asset
func SplitAssetQuote(pair string) (asset, quote string) {
	for i := len(pair) - 1; i >= 0; i-- {
		for _, q := range pairs {
			if i >= len(q)-1 && pair[i-len(q)+1:i+1] == q {
				return pair[:i-len(q)+1], pair[i-len(q)+1:]
			}
		}
	}
	return pair[:len(pair)/2], pair[len(pair)/2:]
}

// ---------------------
// Formatting Functions
// ---------------------

// formatQuantity formats a quantity according to the pair's precision
func formatQuantity(assetsInfo map[string]core.AssetInfo, pair string, value float64) string {
	info, ok := assetsInfo[pair]
	if !ok {
		// Use default precision if asset info not found
		return strconv.FormatFloat(value, 'f', 8, 64)
	}

	// Format according to step size
	step := info.StepSize
	precision := 0
	for step < 1 {
		step *= 10
		precision++
	}

	return strconv.FormatFloat(value, 'f', precision, 64)
}

// formatPrice formats a price according to the pair's precision
func formatPrice(assetsInfo map[string]core.AssetInfo, pair string, value float64) string {
	info, ok := assetsInfo[pair]
	if !ok {
		// Use default precision if asset info not found
		return strconv.FormatFloat(value, 'f', 8, 64)
	}

	// Format according to tick size
	tickSize := info.TickSize
	precision := 0
	for tickSize < 1 {
		tickSize *= 10
		precision++
	}

	return strconv.FormatFloat(value, 'f', precision, 64)
}

// ---------------------
// Validation Functions
// ---------------------

// validateOrder checks if an order quantity is valid for a pair
func validateOrder(assetsInfo map[string]core.AssetInfo, pair string, quantity float64) error {
	info, ok := assetsInfo[pair]
	if !ok {
		return fmt.Errorf("asset info not found for pair: %s", pair)
	}

	if quantity < info.MinQuantity {
		return fmt.Errorf("quantity %f is less than minimum quantity %f", quantity, info.MinQuantity)
	}

	if quantity > info.MaxQuantity {
		return fmt.Errorf("quantity %f is greater than maximum quantity %f", quantity, info.MaxQuantity)
	}

	// Ensure quantity is multiple of step size
	remainder := quantity - info.MinQuantity
	steps := remainder / info.StepSize
	expectedQuantity := info.MinQuantity + (steps * info.StepSize)

	diff := quantity - expectedQuantity
	if diff > 0.000000001 || diff < -0.000000001 {
		return fmt.Errorf("quantity %f is not a multiple of step size %f", quantity, info.StepSize)
	}

	return nil
}

// ---------------------
// Utility Functions
// ---------------------

// setupBackoffRetry creates a backoff with sensible defaults
func setupBackoffRetry() *backoff.Backoff {
	return &backoff.Backoff{
		Min: 100 * time.Millisecond,
		Max: 1 * time.Second,
	}
}

// parseFilterFloat safely parses a float value from a filter map with error handling
func parseFilterFloat(filter map[string]any, key string) (float64, error) {
	value, ok := filter[key]
	if !ok {
		return 0, fmt.Errorf("key %s not found in filter", key)
	}

	strValue, ok := value.(string)
	if !ok {
		return 0, fmt.Errorf("key %s is not a string", key)
	}

	floatValue, err := strconv.ParseFloat(strValue, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse %s as float: %w", key, err)
	}

	return floatValue, nil
}

// ---------------------
// Asset Info Management
// ---------------------

// updateAssetInfoQuantity applies quantity-related parameters to an asset info
func updateAssetInfoQuantity(assetInfo *core.AssetInfo, minQuantity, maxQuantity, stepSize float64) error {
	if err := assetInfo.ChangeMinQuantity(minQuantity); err != nil {
		return fmt.Errorf("failed to update minQuantity: %w", err)
	}

	if err := assetInfo.ChangeMaxQuantity(maxQuantity); err != nil {
		return fmt.Errorf("failed to update maxQuantity: %w", err)
	}

	if err := assetInfo.ChangeStepSize(stepSize); err != nil {
		return fmt.Errorf("failed to update stepSize: %w", err)
	}

	return nil
}

// updateAssetInfoPrice applies price-related parameters to an asset info
func updateAssetInfoPrice(assetInfo *core.AssetInfo, minPrice, maxPrice, tickSize float64) error {
	if err := assetInfo.ChangeMinPrice(minPrice); err != nil {
		return fmt.Errorf("failed to update minPrice: %w", err)
	}

	if err := assetInfo.ChangeMaxPrice(maxPrice); err != nil {
		return fmt.Errorf("failed to update maxPrice: %w", err)
	}

	if err := assetInfo.ChangeTickSize(tickSize); err != nil {
		return fmt.Errorf("failed to update tickSize: %w", err)
	}

	return nil
}

// ---------------------
// Filter Processing
// ---------------------

// processFilter extracts trading limits from symbol filters
func processFilter(filterType string, filter map[string]any, assetInfo *core.AssetInfo) error {
	switch binance.SymbolFilterType(filterType) {
	case binance.SymbolFilterTypeLotSize:
		return processLotSizeFilter(filter, assetInfo)
	case binance.SymbolFilterTypePriceFilter:
		return processPriceFilter(filter, assetInfo)
	default:
		return fmt.Errorf("unknown binance future symbol filter type")
	}
}

// processLotSizeFilter extracts and applies quantity-related parameters from a LotSize filter
func processLotSizeFilter(filter map[string]any, assetInfo *core.AssetInfo) error {
	minQuantity, err := parseFilterFloat(filter, "minQty")
	if err != nil {
		return fmt.Errorf("failed to parse minQty: %w", err)
	}

	maxQuantity, err := parseFilterFloat(filter, "maxQty")
	if err != nil {
		return fmt.Errorf("failed to parse maxQty: %w", err)
	}

	stepSize, err := parseFilterFloat(filter, "stepSize")
	if err != nil {
		return fmt.Errorf("failed to parse stepSize: %w", err)
	}

	return updateAssetInfoQuantity(assetInfo, minQuantity, maxQuantity, stepSize)
}

// processPriceFilter extracts and applies price-related parameters from a PriceFilter filter
func processPriceFilter(filter map[string]any, assetInfo *core.AssetInfo) error {
	minPrice, err := parseFilterFloat(filter, "minPrice")
	if err != nil {
		return fmt.Errorf("failed to parse minPrice: %w", err)
	}

	maxPrice, err := parseFilterFloat(filter, "maxPrice")
	if err != nil {
		return fmt.Errorf("failed to parse maxPrice: %w", err)
	}

	tickSize, err := parseFilterFloat(filter, "tickSize")
	if err != nil {
		return fmt.Errorf("failed to parse tickSize: %w", err)
	}

	return updateAssetInfoPrice(assetInfo, minPrice, maxPrice, tickSize)
}

// ---------------------
// Conversion Functions
// ---------------------

// createAssetInfo creates an AssetInfo object from symbol information
func createAssetInfo[T binance.Symbol | futures.Symbol](info T) (core.AssetInfo, error) {
	var baseAsset, quoteAsset string
	var quotePrecision, baseAssetPrecision int
	var filters []map[string]any

	switch v := any(info).(type) {
	case binance.Symbol:
		baseAsset = v.BaseAsset
		quoteAsset = v.QuoteAsset
		quotePrecision = v.QuotePrecision
		baseAssetPrecision = v.BaseAssetPrecision
		filters = v.Filters

	case futures.Symbol:
		baseAsset = v.BaseAsset
		quoteAsset = v.QuoteAsset
		quotePrecision = v.QuotePrecision
		baseAssetPrecision = v.BaseAssetPrecision
		filters = v.Filters
	}

	// Initialize basic asset info
	assetInfo, err := core.NewAssetInfo(
		baseAsset,
		quoteAsset,
		0, // These values will be filled from filters
		0,
		0,
		0,
		0,
		0,
		quotePrecision,
		baseAssetPrecision,
	)
	if err != nil {
		return core.AssetInfo{}, fmt.Errorf("create asset info error: %w", err)
	}

	// Process trading filters to extract limits and precisions
	for _, filter := range filters {
		if typ, ok := filter["filterType"]; ok {
			processFilter(typ.(string), filter, &assetInfo)
		}
	}

	return assetInfo, nil
}

// convertOrder converts a Binance order to a core.Order
func convertOrder[T *futures.Order | *binance.Order](order T) core.Order {
	var (
		cost, quantity, originQuantity, price float64
		orderID, tm, updateTime               int64
		symbol, side, typ, status             string
	)

	// Extract data based on the concrete type
	switch v := any(order).(type) {
	// Extract data from futures.Order
	case *futures.Order:
		cost, _ = strconv.ParseFloat(v.CumQuote, 64)
		quantity, _ = strconv.ParseFloat(v.ExecutedQuantity, 64)
		originQuantity, _ = strconv.ParseFloat(v.OrigQuantity, 64)
		price, _ = strconv.ParseFloat(v.Price, 64)
		orderID = v.OrderID
		symbol = v.Symbol
		tm = v.Time
		updateTime = v.UpdateTime
		typ = string(v.Type)
		status = string(v.Status)
		side = string(v.Side)

	// Extract data from binance.Order
	case *binance.Order:
		cost, _ = strconv.ParseFloat(v.CummulativeQuoteQuantity, 64)
		quantity, _ = strconv.ParseFloat(v.ExecutedQuantity, 64)
		originQuantity, _ = strconv.ParseFloat(v.OrigQuantity, 64)
		price, _ = strconv.ParseFloat(v.Price, 64)
		orderID = v.OrderID
		symbol = v.Symbol
		tm = v.Time
		updateTime = v.UpdateTime
		typ = string(v.Type)
		status = string(v.Status)
		side = string(v.Side)
	}

	// Calculate effective price if we have valid cost and quantity
	if cost > 0 && quantity > 0 {
		price = cost / quantity
	} else {
		quantity = originQuantity
	}

	// Convert timestamps to proper time.Time objects
	createdAt := time.Unix(0, tm*int64(time.Millisecond))
	updatedAt := time.Unix(0, updateTime*int64(time.Millisecond))

	// Return the standardized core.Order
	return core.Order{
		ExchangeID: orderID,
		Pair:       symbol,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
		Side:       core.SideType(side),
		Type:       core.OrderType(typ),
		Status:     core.OrderStatusType(status),
		Price:      price,
		Quantity:   quantity,
	}
}
