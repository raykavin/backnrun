package binance

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/raykavin/backnrun/pkg/core"

	"github.com/adshao/go-binance/v2"

	"github.com/rodrigo-brito/ninjabot/tools/log"
)

// Spot represents the Binance spot market client
type Spot struct {
	ctx              context.Context
	client           *binance.Client
	assetsInfo       map[string]core.AssetInfo
	heikinAshi       bool
	metadataFetchers []MetadataFetcher
}

// SpotOption is a function that configures a Spot client
type SpotOption func(*Spot)

// WithCredentials sets the API credentials for the Spot client
func WithCredentials(key, secret string) SpotOption {
	return func(s *Spot) {
		s.client = binance.NewClient(key, secret)
	}
}

// WithHeikinAshiCandles enables Heikin Ashi candle conversion
func WithHeikinAshiCandles() SpotOption {
	return func(s *Spot) {
		s.heikinAshi = true
	}
}

// WithMetadataFetcher adds a function for fetching additional candle metadata
func WithMetadataFetcher(fetcher MetadataFetcher) SpotOption {
	return func(s *Spot) {
		s.metadataFetchers = append(s.metadataFetchers, fetcher)
	}
}

// WithTestNet enables the Binance testnet
func WithTestNet() SpotOption {
	return func(_ *Spot) {
		binance.UseTestnet = true
	}
}

// WithCustomMainAPIEndpoint sets custom endpoints for the Binance Main API
func WithCustomMainAPIEndpoint(apiURL, wsURL, combinedURL string) SpotOption {
	if apiURL == "" || wsURL == "" || combinedURL == "" {
		log.Fatal("missing url parameters for custom endpoint configuration")
	}

	return func(_ *Spot) {
		binance.BaseAPIMainURL = apiURL
		binance.BaseWsMainURL = wsURL
		binance.BaseCombinedMainURL = combinedURL
	}
}

// WithCustomTestnetAPIEndpoint sets custom endpoints for the Binance Testnet API
func WithCustomTestnetAPIEndpoint(apiURL, wsURL, combinedURL string) SpotOption {
	if apiURL == "" || wsURL == "" || combinedURL == "" {
		log.Fatal("missing url parameters for custom endpoint configuration")
	}

	return func(_ *Spot) {
		binance.BaseAPITestnetURL = apiURL
		binance.BaseWsTestnetURL = wsURL
		binance.BaseCombinedTestnetURL = combinedURL
	}
}

// NewSpot creates a new Binance spot exchange client
func NewSpot(ctx context.Context, options ...SpotOption) (*Spot, error) {
	binance.WebsocketKeepalive = true

	spot := &Spot{
		ctx:              ctx,
		client:           binance.NewClient("", ""),
		assetsInfo:       make(map[string]core.AssetInfo),
		metadataFetchers: make([]MetadataFetcher, 0),
	}

	// Apply options
	for _, option := range options {
		option(spot)
	}

	// Test connection
	err := spot.client.NewPingService().Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("binance ping fail: %w", err)
	}

	// Get exchange info
	exchangeInfo, err := spot.client.NewExchangeInfoService().Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange info: %w", err)
	}

	// Initialize with orders precision and assets limits
	for _, info := range exchangeInfo.Symbols {
		assetInfo := core.AssetInfo{
			BaseAsset:          info.BaseAsset,
			QuoteAsset:         info.QuoteAsset,
			BaseAssetPrecision: info.BaseAssetPrecision,
			QuotePrecision:     info.QuotePrecision,
		}

		for _, filter := range info.Filters {
			if typ, ok := filter["filterType"]; ok {
				if typ == string(binance.SymbolFilterTypeLotSize) {
					assetInfo.MinQuantity, _ = strconv.ParseFloat(filter["minQty"].(string), 64)
					assetInfo.MaxQuantity, _ = strconv.ParseFloat(filter["maxQty"].(string), 64)
					assetInfo.StepSize, _ = strconv.ParseFloat(filter["stepSize"].(string), 64)
				}

				if typ == string(binance.SymbolFilterTypePriceFilter) {
					assetInfo.MinPrice, _ = strconv.ParseFloat(filter["minPrice"].(string), 64)
					assetInfo.MaxPrice, _ = strconv.ParseFloat(filter["maxPrice"].(string), 64)
					assetInfo.TickSize, _ = strconv.ParseFloat(filter["tickSize"].(string), 64)
				}
			}
		}

		spot.assetsInfo[info.Symbol] = assetInfo
	}

	log.Info("[SETUP] Using Binance Spot exchange")
	return spot, nil
}

// LastQuote gets the latest price for a pair
func (s *Spot) LastQuote(ctx context.Context, pair string) (float64, error) {
	candles, err := s.CandlesByLimit(ctx, pair, "1m", 1)
	if err != nil || len(candles) < 1 {
		return 0, err
	}
	return candles[0].Close, nil
}

// AssetsInfo returns information about an asset
func (s *Spot) AssetsInfo(pair string) core.AssetInfo {
	return s.assetsInfo[pair]
}

// formatQuantity formats a quantity according to the pair's precision
func (s *Spot) formatQuantity(pair string, value float64) string {
	return formatQuantity(s.assetsInfo, pair, value)
}

// formatPrice formats a price according to the pair's precision
func (s *Spot) formatPrice(pair string, value float64) string {
	return formatPrice(s.assetsInfo, pair, value)
}

// validate checks if an order quantity is valid for a pair
func (s *Spot) validate(pair string, quantity float64) error {
	return validateOrder(s.assetsInfo, pair, quantity)
}

// CreateOrderOCO creates an OCO (One-Cancels-the-Other) order
func (s *Spot) CreateOrderOCO(side core.SideType, pair string,
	quantity, price, stop, stopLimit float64) ([]core.Order, error) {

	// Validate quantity
	err := s.validate(pair, quantity)
	if err != nil {
		return nil, err
	}

	// Create OCO order
	ocoOrder, err := s.client.NewCreateOCOService().
		Side(binance.SideType(side)).
		Quantity(s.formatQuantity(pair, quantity)).
		Price(s.formatPrice(pair, price)).
		StopPrice(s.formatPrice(pair, stop)).
		StopLimitPrice(s.formatPrice(pair, stopLimit)).
		StopLimitTimeInForce(binance.TimeInForceTypeGTC).
		Symbol(pair).
		Do(s.ctx)
	if err != nil {
		return nil, err
	}

	// Process response
	orders := make([]core.Order, 0, len(ocoOrder.Orders))
	for _, order := range ocoOrder.OrderReports {
		price, _ := strconv.ParseFloat(order.Price, 64)
		quantity, _ := strconv.ParseFloat(order.OrigQuantity, 64)
		item := core.Order{
			ExchangeID: order.OrderID,
			CreatedAt:  time.Unix(0, ocoOrder.TransactionTime*int64(time.Millisecond)),
			UpdatedAt:  time.Unix(0, ocoOrder.TransactionTime*int64(time.Millisecond)),
			Pair:       pair,
			Side:       core.SideType(order.Side),
			Type:       core.OrderType(order.Type),
			Status:     core.OrderStatusType(order.Status),
			Price:      price,
			Quantity:   quantity,
			GroupID:    &order.OrderListID,
		}

		if item.Type == core.OrderTypeStopLossLimit || item.Type == core.OrderTypeStopLoss {
			item.Stop = &stop
		}

		orders = append(orders, item)
	}

	return orders, nil
}

// CreateOrderStop creates a stop-loss order
func (s *Spot) CreateOrderStop(pair string, quantity, limit float64) (core.Order, error) {
	err := s.validate(pair, quantity)
	if err != nil {
		return core.Order{}, err
	}

	order, err := s.client.NewCreateOrderService().Symbol(pair).
		Type(binance.OrderTypeStopLoss).
		TimeInForce(binance.TimeInForceTypeGTC).
		Side(binance.SideTypeSell).
		Quantity(s.formatQuantity(pair, quantity)).
		Price(s.formatPrice(pair, limit)).
		Do(s.ctx)
	if err != nil {
		return core.Order{}, err
	}

	price, _ := strconv.ParseFloat(order.Price, 64)
	quantity, _ = strconv.ParseFloat(order.OrigQuantity, 64)

	return core.Order{
		ExchangeID: order.OrderID,
		CreatedAt:  time.Unix(0, order.TransactTime*int64(time.Millisecond)),
		UpdatedAt:  time.Unix(0, order.TransactTime*int64(time.Millisecond)),
		Pair:       pair,
		Side:       core.SideType(order.Side),
		Type:       core.OrderType(order.Type),
		Status:     core.OrderStatusType(order.Status),
		Price:      price,
		Quantity:   quantity,
	}, nil
}

// CreateOrderLimit creates a limit order
func (s *Spot) CreateOrderLimit(side core.SideType, pair string,
	quantity, limit float64) (core.Order, error) {

	err := s.validate(pair, quantity)
	if err != nil {
		return core.Order{}, err
	}

	order, err := s.client.NewCreateOrderService().
		Symbol(pair).
		Type(binance.OrderTypeLimit).
		TimeInForce(binance.TimeInForceTypeGTC).
		Side(binance.SideType(side)).
		Quantity(s.formatQuantity(pair, quantity)).
		Price(s.formatPrice(pair, limit)).
		Do(s.ctx)
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
		CreatedAt:  time.Unix(0, order.TransactTime*int64(time.Millisecond)),
		UpdatedAt:  time.Unix(0, order.TransactTime*int64(time.Millisecond)),
		Pair:       pair,
		Side:       core.SideType(order.Side),
		Type:       core.OrderType(order.Type),
		Status:     core.OrderStatusType(order.Status),
		Price:      price,
		Quantity:   quantity,
	}, nil
}

// CreateOrderMarket creates a market order
func (s *Spot) CreateOrderMarket(side core.SideType, pair string, quantity float64) (core.Order, error) {
	err := s.validate(pair, quantity)
	if err != nil {
		return core.Order{}, err
	}

	order, err := s.client.NewCreateOrderService().
		Symbol(pair).
		Type(binance.OrderTypeMarket).
		Side(binance.SideType(side)).
		Quantity(s.formatQuantity(pair, quantity)).
		NewOrderRespType(binance.NewOrderRespTypeFULL).
		Do(s.ctx)
	if err != nil {
		return core.Order{}, err
	}

	cost, err := strconv.ParseFloat(order.CummulativeQuoteQuantity, 64)
	if err != nil {
		return core.Order{}, err
	}

	quantity, err = strconv.ParseFloat(order.ExecutedQuantity, 64)
	if err != nil {
		return core.Order{}, err
	}

	return core.Order{
		ExchangeID: order.OrderID,
		CreatedAt:  time.Unix(0, order.TransactTime*int64(time.Millisecond)),
		UpdatedAt:  time.Unix(0, order.TransactTime*int64(time.Millisecond)),
		Pair:       order.Symbol,
		Side:       core.SideType(order.Side),
		Type:       core.OrderType(order.Type),
		Status:     core.OrderStatusType(order.Status),
		Price:      cost / quantity,
		Quantity:   quantity,
	}, nil
}

// CreateOrderMarketQuote creates a market order with quote quantity
func (s *Spot) CreateOrderMarketQuote(side core.SideType, pair string, quantity float64) (core.Order, error) {
	err := s.validate(pair, quantity)
	if err != nil {
		return core.Order{}, err
	}

	order, err := s.client.NewCreateOrderService().
		Symbol(pair).
		Type(binance.OrderTypeMarket).
		Side(binance.SideType(side)).
		QuoteOrderQty(s.formatQuantity(pair, quantity)).
		NewOrderRespType(binance.NewOrderRespTypeFULL).
		Do(s.ctx)
	if err != nil {
		return core.Order{}, err
	}

	cost, err := strconv.ParseFloat(order.CummulativeQuoteQuantity, 64)
	if err != nil {
		return core.Order{}, err
	}

	quantity, err = strconv.ParseFloat(order.ExecutedQuantity, 64)
	if err != nil {
		return core.Order{}, err
	}

	return core.Order{
		ExchangeID: order.OrderID,
		CreatedAt:  time.Unix(0, order.TransactTime*int64(time.Millisecond)),
		UpdatedAt:  time.Unix(0, order.TransactTime*int64(time.Millisecond)),
		Pair:       order.Symbol,
		Side:       core.SideType(order.Side),
		Type:       core.OrderType(order.Type),
		Status:     core.OrderStatusType(order.Status),
		Price:      cost / quantity,
		Quantity:   quantity,
	}, nil
}

// Cancel cancels an order
func (s *Spot) Cancel(order core.Order) error {
	_, err := s.client.NewCancelOrderService().
		Symbol(order.Pair).
		OrderID(order.ExchangeID).
		Do(s.ctx)
	return err
}

// Orders gets a list of orders for a pair
func (s *Spot) Orders(pair string, limit int) ([]core.Order, error) {
	result, err := s.client.NewListOrdersService().
		Symbol(pair).
		Limit(limit).
		Do(s.ctx)

	if err != nil {
		return nil, err
	}

	orders := make([]core.Order, 0, len(result))
	for _, order := range result {
		orders = append(orders, convertBinanceOrder(order))
	}
	return orders, nil
}

// Order gets a specific order by ID
func (s *Spot) Order(pair string, id int64) (core.Order, error) {
	order, err := s.client.NewGetOrderService().
		Symbol(pair).
		OrderID(id).
		Do(s.ctx)

	if err != nil {
		return core.Order{}, err
	}

	return convertBinanceOrder(order), nil
}

// convertBinanceOrder converts a Binance order to a core.Order
func convertBinanceOrder(order *binance.Order) core.Order {
	var price float64
	cost, _ := strconv.ParseFloat(order.CummulativeQuoteQuantity, 64)
	quantity, _ := strconv.ParseFloat(order.ExecutedQuantity, 64)
	if cost > 0 && quantity > 0 {
		price = cost / quantity
	} else {
		price, _ = strconv.ParseFloat(order.Price, 64)
		quantity, _ = strconv.ParseFloat(order.OrigQuantity, 64)
	}

	return core.Order{
		ExchangeID: order.OrderID,
		Pair:       order.Symbol,
		CreatedAt:  time.Unix(0, order.Time*int64(time.Millisecond)),
		UpdatedAt:  time.Unix(0, order.UpdateTime*int64(time.Millisecond)),
		Side:       core.SideType(order.Side),
		Type:       core.OrderType(order.Type),
		Status:     core.OrderStatusType(order.Status),
		Price:      price,
		Quantity:   quantity,
	}
}

// Account gets the account information
func (s *Spot) Account() (core.Account, error) {
	acc, err := s.client.NewGetAccountService().Do(s.ctx)
	if err != nil {
		return core.Account{}, err
	}

	balances := make([]core.Balance, 0, len(acc.Balances))
	for _, balance := range acc.Balances {
		free, err := strconv.ParseFloat(balance.Free, 64)
		if err != nil {
			return core.Account{}, err
		}
		locked, err := strconv.ParseFloat(balance.Locked, 64)
		if err != nil {
			return core.Account{}, err
		}

		// Skip zero balances for cleaner results
		if free == 0 && locked == 0 {
			continue
		}

		balances = append(balances, core.Balance{
			Asset: balance.Asset,
			Free:  free,
			Lock:  locked,
		})
	}

	return core.Account{
		Balances: balances,
	}, nil
}

// Position gets the current position for a pair
func (s *Spot) Position(pair string) (asset, quote float64, err error) {
	assetTick, quoteTick := SplitAssetQuote(pair)
	acc, err := s.Account()
	if err != nil {
		return 0, 0, err
	}

	assetBalance, quoteBalance := acc.GetBalance(assetTick, quoteTick)

	return assetBalance.Free + assetBalance.Lock, quoteBalance.Free + quoteBalance.Lock, nil
}

// CandlesSubscription subscribes to candle updates for a pair
func (s *Spot) CandlesSubscription(ctx context.Context, pair, period string) (chan core.Candle, chan error) {
	candleChan := make(chan core.Candle)
	errChan := make(chan error)
	heikinAshi := core.NewHeikinAshi()
	backoff := setupBackoffRetry()

	go func() {
		for {
			done, _, err := binance.WsKlineServe(pair, period, func(event *binance.WsKlineEvent) {
				backoff.Reset()
				candle := convertWsKlineToCandle(pair, event.Kline)

				if candle.Complete && s.heikinAshi {
					candle = candle.ToHeikinAshi(heikinAshi)
				}

				if candle.Complete {
					// Fetch additional data if needed
					for _, fetcher := range s.metadataFetchers {
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
func (s *Spot) CandlesByLimit(ctx context.Context, pair, period string, limit int) ([]core.Candle, error) {
	klineService := s.client.NewKlinesService()
	heikinAshi := core.NewHeikinAshi()

	data, err := klineService.Symbol(pair).
		Interval(period).
		Limit(limit + 1). // +1 to discard the last incomplete candle
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

		candle := convertKlineToCandle(pair, *d)

		if s.heikinAshi {
			candle = candle.ToHeikinAshi(heikinAshi)
		}

		candles = append(candles, candle)
	}

	return candles, nil
}

// CandlesByPeriod gets candles for a pair within a time range
func (s *Spot) CandlesByPeriod(ctx context.Context, pair, period string,
	start, end time.Time) ([]core.Candle, error) {

	klineService := s.client.NewKlinesService()
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
		candle := convertKlineToCandle(pair, *d)

		if s.heikinAshi {
			candle = candle.ToHeikinAshi(heikinAshi)
		}

		candles = append(candles, candle)
	}

	return candles, nil
}

// convertKlineToCandle converts a Binance kline to a core.Candle
func convertKlineToCandle(pair string, k binance.Kline) core.Candle {
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

// convertWsKlineToCandle converts a Binance websocket kline to a core.Candle
func convertWsKlineToCandle(pair string, k binance.WsKline) core.Candle {
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
