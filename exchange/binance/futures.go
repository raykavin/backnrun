package binance

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/raykavin/backnrun/core"

	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/common"
	"github.com/adshao/go-binance/v2/futures"
)

// Constants

// MarginType represents the margin type for futures
type MarginType = futures.MarginType

const (
	// MarginTypeIsolated represents isolated margin type
	MarginTypeIsolated MarginType = "ISOLATED"

	// MarginTypeCrossed represents cross margin type
	MarginTypeCrossed MarginType = "CROSSED"

	// ErrNoNeedChangeMarginType is returned when margin type change is not needed
	ErrNoNeedChangeMarginType int64 = -4046
)

// Types

// PairOption represents configuration for a specific trading pair
type PairOption struct {
	Pair       string
	Leverage   int
	MarginType futures.MarginType
}

// Futures represents the Binance futures market client
type Futures struct {
	client           *futures.Client
	assetsInfo       map[string]core.AssetInfo
	heikinAshi       bool
	metadataFetchers []MetadataFetcher
	pairOptions      []PairOption
}

// FuturesOption is a function that configures a Futures client
type FuturesOption func(*Futures)

// Option Functions

// WithFuturesHeikinAshiCandles enables Heikin Ashi candle conversion for futures
func WithFuturesHeikinAshiCandles() FuturesOption {
	return func(f *Futures) {
		f.heikinAshi = true
	}
}

// WithFuturesCredentials sets the API credentials for the Futures client
func WithFuturesCredentials(key, secret string) FuturesOption {
	return func(f *Futures) {
		f.client = futures.NewClient(key, secret)
	}
}

// WithFuturesLeverage sets the leverage for a specific trading pair
func WithFuturesLeverage(pair string, leverage int, marginType MarginType) FuturesOption {
	return func(f *Futures) {
		f.pairOptions = append(f.pairOptions, PairOption{
			Pair:       strings.ToUpper(pair),
			Leverage:   leverage,
			MarginType: marginType,
		})
	}
}

// WithFuturesMetadataFetcher adds a function for fetching additional candle metadata
func WithFuturesMetadataFetcher(fetcher MetadataFetcher) FuturesOption {
	return func(f *Futures) {
		f.metadataFetchers = append(f.metadataFetchers, fetcher)
	}
}

// Constructor Functions

// NewFutures creates a new Binance futures exchange client
func NewFutures(ctx context.Context, options ...FuturesOption) (*Futures, error) {
	binance.WebsocketKeepalive = true

	// Initialize futures struct with default values
	futures := &Futures{
		client:           futures.NewClient("", ""),
		assetsInfo:       make(map[string]core.AssetInfo),
		metadataFetchers: make([]MetadataFetcher, 0),
		pairOptions:      make([]PairOption, 0),
	}

	// Apply custom options
	for _, option := range options {
		option(futures)
	}

	// Validate connection and initialize exchange data
	if err := futures.validateConnection(ctx); err != nil {
		return nil, err
	}

	// Configure pairs with leverage and margin settings
	if err := futures.configurePairs(ctx); err != nil {
		return nil, err
	}

	// Load exchange information and initialize asset data
	if err := futures.initializeAssetInfo(ctx); err != nil {
		return nil, err
	}

	return futures, nil
}

// Utility Functions

// formatQuantity formats a quantity according to the pair's precision
func (f *Futures) formatQuantity(pair string, value float64) string {
	return formatQuantity(f.assetsInfo, pair, value)
}

// formatPrice formats a price according to the pair's precision
func (f *Futures) formatPrice(pair string, value float64) string {
	return formatPrice(f.assetsInfo, pair, value)
}

// validate checks if an order quantity is valid for a pair
func (f *Futures) validate(pair string, quantity float64) error {
	return validateOrder(f.assetsInfo, pair, quantity)
}

// Initialization Methods

// validateConnection tests the connection to the Binance Futures API
func (f *Futures) validateConnection(ctx context.Context) error {
	err := f.client.NewPingService().Do(ctx)
	if err != nil {
		return fmt.Errorf("binance futures ping fail: %w", err)
	}
	return nil
}

// configurePairs sets leverage and margin type for all configured trading pairs
func (f *Futures) configurePairs(ctx context.Context) error {
	for _, option := range f.pairOptions {
		// Set leverage for pair
		_, err := f.client.NewChangeLeverageService().
			Symbol(option.Pair).
			Leverage(option.Leverage).
			Do(ctx)
		if err != nil {
			return fmt.Errorf("failed to set leverage for %s: %w", option.Pair, err)
		}

		// Set margin type for pair
		err = f.client.NewChangeMarginTypeService().
			Symbol(option.Pair).
			MarginType(option.MarginType).
			Do(ctx)
		if err != nil {
			// Ignore "no need to change" error
			if apiError, ok := err.(*common.APIError); !ok || apiError.Code != ErrNoNeedChangeMarginType {
				return fmt.Errorf("failed to set margin type for %s: %w", option.Pair, err)
			}
		}
	}
	return nil
}

// initializeAssetInfo fetches exchange information and initializes asset data
func (f *Futures) initializeAssetInfo(ctx context.Context) error {
	// Get exchange info
	exchangeInfo, err := f.client.NewExchangeInfoService().Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to get futures exchange info: %w", err)
	}

	// Process exchange information for each trading symbol
	for _, info := range exchangeInfo.Symbols {
		assetInfo, err := createAssetInfo(info)
		if err != nil {
			return err
		}

		f.assetsInfo[info.Symbol] = assetInfo
	}

	return nil
}

// API Methods

// LastQuote gets the latest price for a pair
func (f *Futures) LastQuote(ctx context.Context, pair string) (float64, error) {
	candles, err := f.CandlesByLimit(ctx, pair, "1m", 1)
	if err != nil || len(candles) < 1 {
		return 0, err
	}
	return candles[0].Close, nil
}

// AssetsInfo returns information about an asset
func (f *Futures) AssetsInfo(pair string) (core.AssetInfo, error) {
	if val, ok := f.assetsInfo[pair]; ok {
		return val, nil
	}

	return core.AssetInfo{}, fmt.Errorf("asset info not found in binance futures")
}

// Order Management Methods

// CreateOrderOCO creates an OCO (One-Cancels-the-Other) order
// This is not implemented in futures
func (f *Futures) CreateOrderOCO(_ context.Context, _ core.SideType, _ string, _, _, _, _ float64) ([]core.Order, error) {
	return nil, fmt.Errorf("OCO orders not supported in futures market")
}

// CreateOrderStop creates a stop-loss order
func (f *Futures) CreateOrderStop(ctx context.Context, pair string, quantity, limit float64) (core.Order, error) {
	err := f.validate(pair, quantity)
	if err != nil {
		return core.Order{}, err
	}

	order, err := f.client.NewCreateOrderService().Symbol(pair).
		Type(futures.OrderTypeStopMarket).
		TimeInForce(futures.TimeInForceTypeGTC).
		Side(futures.SideTypeSell).
		Quantity(f.formatQuantity(pair, quantity)).
		Price(f.formatPrice(pair, limit)).
		Do(ctx)

	if err != nil {
		return core.Order{}, err
	}

	price, _ := strconv.ParseFloat(order.Price, 64)
	quantity, _ = strconv.ParseFloat(order.OrigQuantity, 64)

	return core.Order{
		ExchangeID: order.OrderID,
		CreatedAt:  time.Unix(0, order.UpdateTime*int64(time.Millisecond)),
		UpdatedAt:  time.Unix(0, order.UpdateTime*int64(time.Millisecond)),
		Pair:       pair,
		Side:       core.SideType(order.Side),
		Type:       core.OrderType(order.Type),
		Status:     core.OrderStatusType(order.Status),
		Price:      price,
		Quantity:   quantity,
	}, nil
}

// CreateOrderLimit creates a limit order
func (f *Futures) CreateOrderLimit(ctx context.Context, side core.SideType, pair string,
	quantity, limit float64) (core.Order, error) {

	err := f.validate(pair, quantity)
	if err != nil {
		return core.Order{}, err
	}

	order, err := f.client.NewCreateOrderService().
		Symbol(pair).
		Type(futures.OrderTypeLimit).
		TimeInForce(futures.TimeInForceTypeGTC).
		Side(futures.SideType(side)).
		Quantity(f.formatQuantity(pair, quantity)).
		Price(f.formatPrice(pair, limit)).
		Do(ctx)

	if err != nil {
		return core.Order{}, err
	}

	price, err := strconv.ParseFloat(order.Price, 64)
	if err != nil {
		return core.Order{}, err
	}

	quantity, err = strconv.ParseFloat(order.OrigQuantity, 64)
	if err != nil {
		return core.Order{}, err
	}

	return core.Order{
		ExchangeID: order.OrderID,
		CreatedAt:  time.Unix(0, order.UpdateTime*int64(time.Millisecond)),
		UpdatedAt:  time.Unix(0, order.UpdateTime*int64(time.Millisecond)),
		Pair:       pair,
		Side:       core.SideType(order.Side),
		Type:       core.OrderType(order.Type),
		Status:     core.OrderStatusType(order.Status),
		Price:      price,
		Quantity:   quantity,
	}, nil
}

// CreateOrderMarket creates a market order
func (f *Futures) CreateOrderMarket(ctx context.Context, side core.SideType, pair string, quantity float64) (core.Order, error) {
	err := f.validate(pair, quantity)
	if err != nil {
		return core.Order{}, err
	}

	order, err := f.client.NewCreateOrderService().
		Symbol(pair).
		Type(futures.OrderTypeMarket).
		Side(futures.SideType(side)).
		Quantity(f.formatQuantity(pair, quantity)).
		NewOrderResponseType(futures.NewOrderRespTypeRESULT).
		Do(ctx)

	if err != nil {
		return core.Order{}, err
	}

	cost, err := strconv.ParseFloat(order.CumQuote, 64)
	if err != nil {
		return core.Order{}, err
	}

	quantity, err = strconv.ParseFloat(order.ExecutedQuantity, 64)
	if err != nil {
		return core.Order{}, err
	}

	return core.Order{
		ExchangeID: order.OrderID,
		CreatedAt:  time.Unix(0, order.UpdateTime*int64(time.Millisecond)),
		UpdatedAt:  time.Unix(0, order.UpdateTime*int64(time.Millisecond)),
		Pair:       order.Symbol,
		Side:       core.SideType(order.Side),
		Type:       core.OrderType(order.Type),
		Status:     core.OrderStatusType(order.Status),
		Price:      cost / quantity,
		Quantity:   quantity,
	}, nil
}

// CreateOrderMarketQuote creates a market order with quote quantity
// This is not implemented in futures
func (f *Futures) CreateOrderMarketQuote(_ context.Context, _ core.SideType, _ string, _ float64) (core.Order, error) {
	return core.Order{}, fmt.Errorf("market quote orders not supported in futures market")
}

// Cancel cancels an order
func (f *Futures) Cancel(ctx context.Context, order core.Order) error {
	_, err := f.client.NewCancelOrderService().
		Symbol(order.Pair).
		OrderID(order.ExchangeID).
		Do(ctx)
	return err
}

// Orders gets a list of orders for a pair
func (f *Futures) Orders(ctx context.Context, pair string, limit int) ([]core.Order, error) {
	result, err := f.client.NewListOrdersService().
		Symbol(pair).
		Limit(limit).
		Do(ctx)

	if err != nil {
		return nil, err
	}

	orders := make([]core.Order, 0, len(result))
	for _, order := range result {
		orders = append(orders, convertOrder(order))
	}
	return orders, nil
}

// Order gets a specific order by ID
func (f *Futures) Order(ctx context.Context, pair string, id int64) (core.Order, error) {
	order, err := f.client.NewGetOrderService().
		Symbol(pair).
		OrderID(id).
		Do(ctx)

	if err != nil {
		return core.Order{}, err
	}

	return convertOrder(order), nil
}

// Account Information Methods

// Account gets the account information
func (f *Futures) Account(ctx context.Context) (core.Account, error) {
	acc, err := f.client.NewGetAccountService().Do(ctx)
	if err != nil {
		return core.Account{}, err
	}

	balances := make([]core.Balance, 0)

	// Process positions
	for _, position := range acc.Positions {
		free, err := strconv.ParseFloat(position.PositionAmt, 64)
		if err != nil {
			return core.Account{}, err
		}

		// Skip zero positions
		if free == 0 {
			continue
		}

		leverage, err := strconv.ParseFloat(position.Leverage, 64)
		if err != nil {
			return core.Account{}, err
		}

		// Adjust for short positions
		if position.PositionSide == futures.PositionSideTypeShort {
			free = -free
		}

		asset, _ := SplitAssetQuote(position.Symbol)

		balances = append(balances, core.Balance{
			Asset:    asset,
			Free:     free,
			Leverage: leverage,
		})
	}

	// Process wallet assets
	for _, asset := range acc.Assets {
		free, err := strconv.ParseFloat(asset.WalletBalance, 64)
		if err != nil {
			return core.Account{}, err
		}

		// Skip zero balances
		if free == 0 {
			continue
		}

		balances = append(balances, core.Balance{
			Asset: asset.Asset,
			Free:  free,
		})
	}

	return core.NewAccount(balances)
}

// Position gets the current position for a pair
func (f *Futures) Position(ctx context.Context, pair string) (asset, quote float64, err error) {
	assetTick, quoteTick := SplitAssetQuote(pair)
	acc, err := f.Account(ctx)
	if err != nil {
		return 0, 0, err
	}

	assetBalance, quoteBalance := acc.GetBalance(assetTick, quoteTick)

	return assetBalance.Free + assetBalance.Lock, quoteBalance.Free + quoteBalance.Lock, nil
}

// Candle Methods

// CandlesSubscription subscribes to candle updates for a pair
func (f *Futures) CandlesSubscription(ctx context.Context, pair, period string) (chan core.Candle, chan error) {
	candleChan := make(chan core.Candle)
	errChan := make(chan error)
	heikinAshi := core.NewHeikinAshi()
	backoff := setupBackoffRetry()

	go func() {
		for {
			done, _, err := futures.WsKlineServe(pair, period, func(event *futures.WsKlineEvent) {
				backoff.Reset()
				candle := convertFuturesWsKlineToCandle(pair, event.Kline)

				if candle.Complete && f.heikinAshi {
					candle = candle.ToHeikinAshi(heikinAshi)
				}

				if candle.Complete {
					// Fetch additional data if needed
					for _, fetcher := range f.metadataFetchers {
						key, value := fetcher(pair, candle.Time)
						candle.Metadata[key] = value
					}
				}

				candleChan <- candle

			}, func(err error) {
				errChan <- err
			})

			if err != nil {
				errChan <- err
				close(errChan)
				close(candleChan)
				return
			}

			select {
			case <-ctx.Done():
				close(errChan)
				close(candleChan)
				return
			case <-done:
				time.Sleep(backoff.Duration())
			}
		}
	}()

	return candleChan, errChan
}

// CandlesByLimit gets a number of candles for a pair
func (f *Futures) CandlesByLimit(ctx context.Context, pair, period string, limit int) ([]core.Candle, error) {
	klineService := f.client.NewKlinesService()
	heikinAshi := core.NewHeikinAshi()

	data, err := klineService.Symbol(pair).
		Interval(period).
		Limit(limit + 1). // +1 to account for the incomplete candle
		Do(ctx)

	if err != nil {
		return nil, err
	}

	candles := make([]core.Candle, 0, len(data)-1)
	for i, d := range data {
		// Skip the last candle as it's incomplete
		if i == len(data)-1 {
			break
		}

		candle := convertFuturesKlineToCandle(pair, *d)

		if f.heikinAshi {
			candle = candle.ToHeikinAshi(heikinAshi)
		}

		candles = append(candles, candle)
	}

	return candles, nil
}

// CandlesByPeriod gets candles for a pair within a time range
func (f *Futures) CandlesByPeriod(ctx context.Context, pair, period string,
	start, end time.Time) ([]core.Candle, error) {

	klineService := f.client.NewKlinesService()
	heikinAshi := core.NewHeikinAshi()

	data, err := klineService.Symbol(pair).
		Interval(period).
		StartTime(start.UnixNano() / int64(time.Millisecond)).
		EndTime(end.UnixNano() / int64(time.Millisecond)).
		Do(ctx)

	if err != nil {
		return nil, err
	}

	candles := make([]core.Candle, 0, len(data))
	for _, d := range data {
		candle := convertFuturesKlineToCandle(pair, *d)

		if f.heikinAshi {
			candle = candle.ToHeikinAshi(heikinAshi)
		}

		candles = append(candles, candle)
	}

	return candles, nil
}

// convertFuturesKlineToCandle converts a Binance futures kline to a core.Candle
func convertFuturesKlineToCandle(pair string, k futures.Kline) core.Candle {
	t := time.Unix(0, k.OpenTime*int64(time.Millisecond))
	candle := core.Candle{
		Pair:      pair,
		Time:      t,
		UpdatedAt: t,
		Metadata:  make(map[string]float64),
		Complete:  true,
	}

	candle.Open, _ = strconv.ParseFloat(k.Open, 64)
	candle.Close, _ = strconv.ParseFloat(k.Close, 64)
	candle.High, _ = strconv.ParseFloat(k.High, 64)
	candle.Low, _ = strconv.ParseFloat(k.Low, 64)
	candle.Volume, _ = strconv.ParseFloat(k.Volume, 64)

	return candle
}

// convertFuturesWsKlineToCandle converts a Binance futures websocket kline to a core.Candle
func convertFuturesWsKlineToCandle(pair string, k futures.WsKline) core.Candle {
	t := time.Unix(0, k.StartTime*int64(time.Millisecond))
	candle := core.Candle{
		Pair:      pair,
		Time:      t,
		UpdatedAt: t,
		Metadata:  make(map[string]float64),
		Complete:  k.IsFinal,
	}

	candle.Open, _ = strconv.ParseFloat(k.Open, 64)
	candle.Close, _ = strconv.ParseFloat(k.Close, 64)
	candle.High, _ = strconv.ParseFloat(k.High, 64)
	candle.Low, _ = strconv.ParseFloat(k.Low, 64)
	candle.Volume, _ = strconv.ParseFloat(k.Volume, 64)

	return candle
}
