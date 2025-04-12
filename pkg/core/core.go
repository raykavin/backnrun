package core

import (
	"context"
	"time"
)

type Exchange interface {
	Broker
	Feeder
}

type Feeder interface {
	AssetsInfo(pair string) AssetInfo
	LastQuote(ctx context.Context, pair string) (float64, error)
	CandlesByPeriod(ctx context.Context, pair, period string, start, end time.Time) ([]Candle, error)
	CandlesByLimit(ctx context.Context, pair, period string, limit int) ([]Candle, error)
	CandlesSubscription(ctx context.Context, pair, timeframe string) (chan Candle, chan error)
}

type Broker interface {
	Account() (Account, error)
	Position(pair string) (asset, quote float64, err error)
	Order(pair string, id int64) (Order, error)
	CreateOrderOCO(side SideType, pair string, size, price, stop, stopLimit float64) ([]Order, error)
	CreateOrderLimit(side SideType, pair string, size float64, limit float64) (Order, error)
	CreateOrderMarket(side SideType, pair string, size float64) (Order, error)
	CreateOrderMarketQuote(side SideType, pair string, quote float64) (Order, error)
	CreateOrderStop(pair string, quantity float64, limit float64) (Order, error)
	Cancel(Order) error
}

type Notifier interface {
	Notify(string)
	OnOrder(order Order)
	OnError(err error)
}

type NotifierWithStart interface {
	Notifier
	Start()
}
