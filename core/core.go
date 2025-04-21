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
	AssetsInfo(pair string) (AssetInfo, error)
	LastQuote(ctx context.Context, pair string) (float64, error)
	CandlesByPeriod(ctx context.Context, pair, period string, start, end time.Time) ([]Candle, error)
	CandlesByLimit(ctx context.Context, pair, period string, limit int) ([]Candle, error)
	CandlesSubscription(ctx context.Context, pair, timeframe string) (chan Candle, chan error)
}

type Broker interface {
	Account(ctx context.Context) (Account, error)
	Position(ctx context.Context, pair string) (asset, quote float64, err error)
	Order(ctx context.Context, pair string, id int64) (Order, error)
	CreateOrderOCO(ctx context.Context, side SideType, pair string, size, price, stop, stopLimit float64) ([]Order, error)
	CreateOrderLimit(ctx context.Context, side SideType, pair string, size float64, limit float64) (Order, error)
	CreateOrderMarket(ctx context.Context, side SideType, pair string, size float64) (Order, error)
	CreateOrderMarketQuote(ctx context.Context, side SideType, pair string, quote float64) (Order, error)
	CreateOrderStop(ctx context.Context, pair string, quantity float64, limit float64) (Order, error)
	Cancel(ctx context.Context, order Order) error
}

type Strategy interface {
	// Timeframe is the time interval in which the strategy will be executed. eg: 1h, 1d, 1w
	Timeframe() string
	// WarmupPeriod is the necessary time to wait before executing the strategy, to load data for indicators.
	// This time is measured in the period specified in the `Timeframe` function.
	WarmupPeriod() int
	// Indicators will be executed for each new candle, in order to fill indicators before `OnCandle` function is called.
	Indicators(df *Dataframe) []ChartIndicator
	// OnCandle will be executed for each new candle, after indicators are filled, here you can do your trading logic.
	// OnCandle is executed after the candle close.
	OnCandle(ctx context.Context, df *Dataframe, broker Broker)
}

type HighFrequencyStrategy interface {
	Strategy

	// OnPartialCandle will be executed for each new partial candle, after indicators are filled.
	OnPartialCandle(df *Dataframe, broker Broker)
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
