package exchange

import (
	"context"
	"testing"
	"time"

	"github.com/raykavin/backnrun/core"
	"github.com/raykavin/backnrun/logger/zerolog"

	"github.com/stretchr/testify/require"
)

func getLog() core.Logger {
	l, err := zerolog.New("debug", "2006-01-02 15:04:05", true, false)
	if err != nil {
		panic(err)
	}

	return zerolog.NewAdapter(l.Logger)

}

func TestPaperWallet_ValidateFunds(t *testing.T) {
	t.Run("simple lock limit", func(t *testing.T) {
		wallet := NewPaperWallet(context.Background(), "USDT", getLog(), WithPaperAsset("USDT", 100))
		err := wallet.validateFunds(core.SideTypeBuy, "BTCUSDT", 1, 100, false)
		require.NoError(t, err)
		require.Equal(t, 0.0, wallet.assets["USDT"].Free)
		require.Equal(t, 100.0, wallet.assets["USDT"].Lock)
		require.Equal(t, 0.0, wallet.assets["BTC"].Free)
		require.Equal(t, 0.0, wallet.assets["BTC"].Lock)
	})

	t.Run("simple buy market", func(t *testing.T) {
		wallet := NewPaperWallet(context.Background(), "USDT", getLog(), WithPaperAsset("USDT", 100))
		wallet.lastCandle["BTCUSDT"] = core.Candle{Pair: "BTCUSDT", Close: 100}
		err := wallet.validateFunds(core.SideTypeBuy, "BTCUSDT", 1, 100, true)
		require.NoError(t, err)
		require.Equal(t, 0.0, wallet.assets["USDT"].Free)
		require.Equal(t, 0.0, wallet.assets["USDT"].Lock)
		require.Equal(t, 1.0, wallet.assets["BTC"].Free)
		require.Equal(t, 0.0, wallet.assets["BTC"].Lock)
	})

	t.Run("simple short market", func(t *testing.T) {
		wallet := NewPaperWallet(context.Background(), "USDT", getLog(), WithPaperAsset("USDT", 100))
		wallet.lastCandle["BTCUSDT"] = core.Candle{Pair: "BTCUSDT", Close: 100}
		err := wallet.validateFunds(core.SideTypeSell, "BTCUSDT", 1, 100, true)
		require.NoError(t, err)
		require.Equal(t, 0.0, wallet.assets["USDT"].Free)
		require.Equal(t, 0.0, wallet.assets["USDT"].Lock)
		require.Equal(t, -1.0, wallet.assets["BTC"].Free)
		require.Equal(t, 0.0, wallet.assets["BTC"].Lock)
	})

	t.Run("simple short limit", func(t *testing.T) {
		wallet := NewPaperWallet(context.Background(), "USDT", getLog(), WithPaperAsset("USDT", 100))
		err := wallet.validateFunds(core.SideTypeSell, "BTCUSDT", 1, 100, false)
		require.NoError(t, err)
		require.Equal(t, 0.0, wallet.assets["USDT"].Free)
		require.Equal(t, 100.0, wallet.assets["USDT"].Lock)
		require.Equal(t, 0.0, wallet.assets["BTC"].Free)
		require.Equal(t, 0.0, wallet.assets["BTC"].Lock)
	})

	t.Run("invert position long to short", func(t *testing.T) {
		wallet := NewPaperWallet(context.Background(), "USDT", getLog(), WithPaperAsset("BTC", 1), WithPaperAsset("USDT", 100))
		wallet.avgLongPrice["BTCUSDT"] = 100

		err := wallet.validateFunds(core.SideTypeSell, "BTCUSDT", 2, 100, true)
		require.NoError(t, err)
		require.Equal(t, 0.0, wallet.assets["USDT"].Free)
		require.Equal(t, 0.0, wallet.assets["USDT"].Lock)
		require.Equal(t, -2.0, wallet.assets["BTC"].Free)
		require.Equal(t, 0.0, wallet.assets["BTC"].Lock)
	})

	t.Run("invert position short to long", func(t *testing.T) {
		wallet := NewPaperWallet(context.Background(), "USDT", getLog(), WithPaperAsset("BTC", -1), WithPaperAsset("USDT", 100))
		wallet.avgShortPrice["BTCUSDT"] = 100

		err := wallet.validateFunds(core.SideTypeBuy, "BTCUSDT", 2, 150, true)
		require.NoError(t, err)
		require.Equal(t, 0.0, wallet.assets["USDT"].Free)
		require.Equal(t, 0.0, wallet.assets["USDT"].Lock)
		require.Equal(t, 1.0, wallet.assets["BTC"].Free)
		require.Equal(t, 0.0, wallet.assets["BTC"].Lock)
	})
}

func TestPaperWallet_OrderLimit(t *testing.T) {
	t.Run("normal order", func(t *testing.T) {
		wallet := NewPaperWallet(context.Background(), "USDT", getLog(), WithPaperAsset("USDT", 100))
		order, err := wallet.CreateOrderLimit(context.Background(), core.SideTypeBuy, "BTCUSDT", 1, 100)
		require.NoError(t, err)

		// create order and lock values
		require.Len(t, wallet.orders, 1)
		require.Equal(t, 1.0, order.Quantity)
		require.Equal(t, 100.0, order.Price)
		require.Equal(t, 0.0, wallet.assets["USDT"].Free)
		require.Equal(t, 100.0, wallet.assets["USDT"].Lock)

		// a new candle should execute order and unlock values
		wallet.OnCandle(core.Candle{Pair: "BTCUSDT", Close: 100})
		require.Equal(t, core.OrderStatusTypeFilled, wallet.orders[0].Status)
		require.Equal(t, 0.0, wallet.assets["USDT"].Free)
		require.Equal(t, 0.0, wallet.assets["USDT"].Lock)
		require.Equal(t, 1.0, wallet.assets["BTC"].Free)
		require.Equal(t, 0.0, wallet.assets["BTC"].Lock)
		require.Equal(t, 100.0, wallet.avgLongPrice["BTCUSDT"])

		// try to buy again without funds
		order, err = wallet.CreateOrderLimit(context.Background(), core.SideTypeBuy, "BTCUSDT", 1, 100)
		require.Empty(t, order)
		require.Equal(t, &OrderError{
			Err:      ErrInsufficientFunds,
			Pair:     "BTCUSDT",
			Quantity: 1,
		}, err)

		// try to sell and profit 100 USDT
		order, err = wallet.CreateOrderLimit(context.Background(), core.SideTypeSell, "BTCUSDT", 1, 200)
		require.NoError(t, err)
		require.Len(t, wallet.orders, 2)
		require.Equal(t, 1.0, order.Quantity)
		require.Equal(t, 200.0, order.Price)
		require.Equal(t, 0.0, wallet.assets["USDT"].Free)
		require.Equal(t, 0.0, wallet.assets["USDT"].Lock)
		require.Equal(t, 0.0, wallet.assets["BTC"].Free)
		require.Equal(t, 1.0, wallet.assets["BTC"].Lock)

		// a new candle should execute order and unlock values
		wallet.OnCandle(core.Candle{Pair: "BTCUSDT", Close: 200, High: 200})
		require.Equal(t, core.OrderStatusTypeFilled, wallet.orders[1].Status)
		require.Equal(t, 200.0, wallet.assets["USDT"].Free)
		require.Equal(t, 0.0, wallet.assets["USDT"].Lock)
		require.Equal(t, 0.0, wallet.assets["BTC"].Free)
		require.Equal(t, 0.0, wallet.assets["BTC"].Lock)
	})

	t.Run("multiple pending orders", func(t *testing.T) {
		wallet := NewPaperWallet(context.Background(), "USDT", getLog(), WithPaperAsset("USDT", 100))
		wallet.lastCandle["BTCUSDT"] = core.Candle{Close: 10}

		order, err := wallet.CreateOrderLimit(context.Background(), core.SideTypeBuy, "BTCUSDT", 1, 10)
		require.NoError(t, err)
		require.NotEmpty(t, order)

		require.Equal(t, 90.0, wallet.assets["USDT"].Free)
		require.Equal(t, 10.0, wallet.assets["USDT"].Lock)

		order, err = wallet.CreateOrderLimit(context.Background(), core.SideTypeBuy, "BTCUSDT", 1, 20)
		require.NoError(t, err)
		require.NotEmpty(t, order)

		require.Equal(t, 70.0, wallet.assets["USDT"].Free)
		require.Equal(t, 30.0, wallet.assets["USDT"].Lock)

		order, err = wallet.CreateOrderLimit(context.Background(), core.SideTypeBuy, "BTCUSDT", 1, 50)
		require.NoError(t, err)
		require.NotEmpty(t, order)

		require.Equal(t, 20.0, wallet.assets["USDT"].Free)
		require.Equal(t, 80.0, wallet.assets["USDT"].Lock)

		// should execute two orders and keep one pending
		wallet.OnCandle(core.Candle{Pair: "BTCUSDT", Close: 15})
		require.Equal(t, 20.0, wallet.assets["USDT"].Free)
		require.Equal(t, 10.0, wallet.assets["USDT"].Lock)
		require.Equal(t, 0.0, wallet.assets["BTC"].Lock)
		require.Equal(t, 2.0, wallet.assets["BTC"].Free)
		require.Equal(t, 35.0, wallet.avgLongPrice["BTCUSDT"])
		require.Equal(t, core.OrderStatusTypeNew, wallet.orders[0].Status)
		require.Equal(t, core.OrderStatusTypeFilled, wallet.orders[1].Status)
		require.Equal(t, core.OrderStatusTypeFilled, wallet.orders[2].Status)

		// sell all bitcoin position
		order, err = wallet.CreateOrderLimit(context.Background(), core.SideTypeSell, "BTCUSDT", 2, 40)
		require.NoError(t, err)
		require.NotEmpty(t, order)

		require.Equal(t, 20.0, wallet.assets["USDT"].Free)
		require.Equal(t, 10.0, wallet.assets["USDT"].Lock)
		require.Equal(t, 0.0, wallet.assets["BTC"].Free)
		require.Equal(t, 2.0, wallet.assets["BTC"].Lock)

		wallet.OnCandle(core.Candle{Pair: "BTCUSDT", Close: 50, High: 50})
		require.Equal(t, 0.0, wallet.assets["BTC"].Free)
		require.Equal(t, 0.0, wallet.assets["BTC"].Lock)
		require.Equal(t, 100.0, wallet.assets["USDT"].Free)
		require.Equal(t, 10.0, wallet.assets["USDT"].Lock)

		// execute old buy position
		wallet.OnCandle(core.Candle{Pair: "BTCUSDT", Close: 9, High: 9})
		require.Equal(t, 1.0, wallet.assets["BTC"].Free)
		require.Equal(t, 0.0, wallet.assets["BTC"].Lock)
		require.Equal(t, 100.0, wallet.assets["USDT"].Free)
		require.Equal(t, 0.0, wallet.assets["USDT"].Lock)
		require.Equal(t, 10.0, wallet.avgLongPrice["BTCUSDT"])
	})

	t.Run("cancel buy order before executing", func(t *testing.T) {
		wallet := NewPaperWallet(context.Background(), "USDT", getLog(), WithPaperAsset("USDT", 100))
		order, err := wallet.CreateOrderLimit(context.Background(), core.SideTypeBuy, "BTCUSDT", 1, 100)
		require.NoError(t, err)

		// create order and lock values
		require.Len(t, wallet.orders, 1)
		require.Equal(t, 1.0, order.Quantity)
		require.Equal(t, 100.0, order.Price)
		require.Equal(t, 0.0, wallet.assets["USDT"].Free)
		require.Equal(t, 100.0, wallet.assets["USDT"].Lock)

		// cancel limit order and it should unlock funds
		err = wallet.Cancel(context.Background(), order)
		require.NoError(t, err)

		require.Equal(t, core.OrderStatusTypeCanceled, wallet.orders[0].Status)
		require.Equal(t, 100.0, wallet.assets["USDT"].Free)
		require.Equal(t, 0.0, wallet.assets["USDT"].Lock)
		require.Equal(t, 0.0, wallet.assets["BTC"].Free)
		require.Equal(t, 0.0, wallet.assets["BTC"].Lock)
	})

	t.Run("cancel sell order before executing", func(t *testing.T) {
		wallet := NewPaperWallet(context.Background(), "USDT", getLog(), WithPaperAsset("USDT", 100))
		order, err := wallet.CreateOrderLimit(context.Background(), core.SideTypeSell, "BTCUSDT", 1, 100)
		require.NoError(t, err)

		// create order and lock values
		require.Len(t, wallet.orders, 1)
		require.Equal(t, 1.0, order.Quantity)
		require.Equal(t, 100.0, order.Price)
		require.Equal(t, 0.0, wallet.assets["USDT"].Free)
		require.Equal(t, 100.0, wallet.assets["USDT"].Lock)

		// cancel limit order and it should unlock funds
		err = wallet.Cancel(context.Background(), order)
		require.NoError(t, err)

		require.Equal(t, core.OrderStatusTypeCanceled, wallet.orders[0].Status)
		require.Equal(t, 100.0, wallet.assets["USDT"].Free)
		require.Equal(t, 0.0, wallet.assets["USDT"].Lock)
		require.Equal(t, 0.0, wallet.assets["BTC"].Free)
		require.Equal(t, 0.0, wallet.assets["BTC"].Lock)
	})
}

func TestPaperWallet_OrderMarket(t *testing.T) {
	wallet := NewPaperWallet(context.Background(), "USDT", getLog(), WithPaperAsset("USDT", 100))
	wallet.OnCandle(core.Candle{Pair: "BTCUSDT", Close: 50})
	order, err := wallet.CreateOrderMarket(context.Background(), core.SideTypeBuy, "BTCUSDT", 1)
	require.NoError(t, err)

	// create buy order
	require.Len(t, wallet.orders, 1)
	require.Equal(t, core.OrderStatusTypeFilled, order.Status)
	require.Equal(t, 1.0, order.Quantity)
	require.Equal(t, 50.0, order.Price)
	require.Equal(t, 50.0, wallet.assets["USDT"].Free)
	require.Equal(t, 0.0, wallet.assets["USDT"].Lock)
	require.Equal(t, 1.0, wallet.assets["BTC"].Free)
	require.Equal(t, 0.0, wallet.assets["BTC"].Lock)
	require.Equal(t, 50.0, wallet.avgLongPrice["BTCUSDT"])

	// insufficient funds
	order, err = wallet.CreateOrderMarket(context.Background(), core.SideTypeBuy, "BTCUSDT", 100)
	require.Equal(t, &OrderError{
		Err:      ErrInsufficientFunds,
		Pair:     "BTCUSDT",
		Quantity: 100}, err)
	require.Empty(t, order)

	// sell
	wallet.OnCandle(core.Candle{Pair: "BTCUSDT", Close: 100})
	order, err = wallet.CreateOrderMarket(context.Background(), core.SideTypeSell, "BTCUSDT", 1)
	require.NoError(t, err)
	require.Equal(t, 1.0, order.Quantity)
	require.Equal(t, 100.0, order.Price)
	require.Equal(t, 150.0, wallet.assets["USDT"].Free)
	require.Equal(t, 0.0, wallet.assets["USDT"].Lock)
	require.Equal(t, 0.0, wallet.assets["BTC"].Free)
	require.Equal(t, 0.0, wallet.assets["BTC"].Lock)
	require.Equal(t, 50.0, wallet.avgLongPrice["BTCUSDT"])
}

func TestPaperWallet_OrderOCO(t *testing.T) {
	wallet := NewPaperWallet(context.Background(), "USDT", getLog(), WithPaperAsset("USDT", 50))
	wallet.OnCandle(core.Candle{Pair: "BTCUSDT", Close: 50})
	_, err := wallet.CreateOrderMarket(context.Background(), core.SideTypeBuy, "BTCUSDT", 1)
	require.NoError(t, err)

	orders, err := wallet.CreateOrderOCO(context.Background(), core.SideTypeSell, "BTCUSDT", 1, 100, 40, 39)
	require.NoError(t, err)

	// create buy order
	require.Len(t, wallet.orders, 3)
	require.Equal(t, core.OrderStatusTypeNew, orders[0].Status)
	require.Equal(t, core.OrderStatusTypeNew, orders[1].Status)
	require.Equal(t, 1.0, orders[0].Quantity)
	require.Equal(t, 1.0, orders[1].Quantity)

	require.Equal(t, 0.0, wallet.assets["USDT"].Free)
	require.Equal(t, 0.0, wallet.assets["USDT"].Lock)
	require.Equal(t, 0.0, wallet.assets["BTC"].Free)
	require.Equal(t, 1.0, wallet.assets["BTC"].Lock)

	// insufficient funds
	orders, err = wallet.CreateOrderOCO(context.Background(), core.SideTypeSell, "BTCUSDT", 1, 100, 40, 39)
	require.Equal(t, &OrderError{
		Err:      ErrInsufficientFunds,
		Pair:     "BTCUSDT",
		Quantity: 1}, err)
	require.Nil(t, orders)

	// execute stop and cancel target
	wallet.OnCandle(core.Candle{Pair: "BTCUSDT", Close: 30})
	require.Equal(t, 40.0, wallet.assets["USDT"].Free)
	require.Equal(t, 0.0, wallet.assets["USDT"].Lock)
	require.Equal(t, 0.0, wallet.assets["BTC"].Free)
	require.Equal(t, 0.0, wallet.assets["BTC"].Lock)
	require.Equal(t, wallet.orders[1].Status, core.OrderStatusTypeCanceled)
	require.Equal(t, wallet.orders[2].Status, core.OrderStatusTypeFilled)
}

func TestPaperWallet_Order(t *testing.T) {
	wallet := NewPaperWallet(context.Background(), "USDT", getLog(), WithPaperAsset("USDT", 100))
	expectOrder, err := wallet.CreateOrderMarket(context.Background(), core.SideTypeBuy, "BTCUSDT", 1)
	require.NoError(t, err)
	require.Equal(t, int64(1), expectOrder.ExchangeID)

	order, err := wallet.Order(context.Background(), "BTCUSDT", expectOrder.ExchangeID)
	require.NoError(t, err)
	require.Equal(t, expectOrder, order)
}

func TestPaperWallet_MaxDrawndown(t *testing.T) {
	tt := []struct {
		name   string
		values []AssetValue
		result float64
		start  time.Time
		end    time.Time
	}{
		{
			name: "down only",
			values: []AssetValue{
				{Time: time.Date(2019, time.January, 1, 0, 0, 0, 0, time.UTC), Value: 10},
				{Time: time.Date(2019, time.January, 2, 0, 0, 0, 0, time.UTC), Value: 5},
			},
			result: -0.5,
			start:  time.Date(2019, time.January, 1, 0, 0, 0, 0, time.UTC),
			end:    time.Date(2019, time.January, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "up and down",
			values: []AssetValue{
				{Time: time.Date(2019, time.January, 1, 0, 0, 0, 0, time.UTC), Value: 1},
				{Time: time.Date(2019, time.January, 2, 0, 0, 0, 0, time.UTC), Value: 10},
				{Time: time.Date(2019, time.January, 3, 0, 0, 0, 0, time.UTC), Value: 5},
			},
			result: -0.5,
			start:  time.Date(2019, time.January, 2, 0, 0, 0, 0, time.UTC),
			end:    time.Date(2019, time.January, 3, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "down and up",
			values: []AssetValue{
				{Time: time.Date(2019, time.January, 1, 0, 0, 0, 0, time.UTC), Value: 4},
				{Time: time.Date(2019, time.January, 2, 0, 0, 0, 0, time.UTC), Value: 5},
				{Time: time.Date(2019, time.January, 3, 0, 0, 0, 0, time.UTC), Value: 4},
				{Time: time.Date(2019, time.January, 4, 0, 0, 0, 0, time.UTC), Value: 3},
				{Time: time.Date(2019, time.January, 5, 0, 0, 0, 0, time.UTC), Value: 4},
				{Time: time.Date(2019, time.January, 6, 0, 0, 0, 0, time.UTC), Value: 5},
				{Time: time.Date(2019, time.January, 7, 0, 0, 0, 0, time.UTC), Value: 6},
				{Time: time.Date(2019, time.January, 8, 0, 0, 0, 0, time.UTC), Value: 7},
				{Time: time.Date(2019, time.January, 9, 0, 0, 0, 0, time.UTC), Value: 6},
			},
			result: -0.4,
			start:  time.Date(2019, time.January, 2, 0, 0, 0, 0, time.UTC),
			end:    time.Date(2019, time.January, 4, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "two drawn downs",
			values: []AssetValue{
				{Time: time.Date(2019, time.January, 1, 0, 0, 0, 0, time.UTC), Value: 1},
				{Time: time.Date(2019, time.January, 2, 0, 0, 0, 0, time.UTC), Value: 5},
				{Time: time.Date(2019, time.January, 3, 0, 0, 0, 0, time.UTC), Value: 4},
				{Time: time.Date(2019, time.January, 4, 0, 0, 0, 0, time.UTC), Value: 7},
				{Time: time.Date(2019, time.January, 5, 0, 0, 0, 0, time.UTC), Value: 8},
				{Time: time.Date(2019, time.January, 6, 0, 0, 0, 0, time.UTC), Value: 4},
				{Time: time.Date(2019, time.January, 7, 0, 0, 0, 0, time.UTC), Value: 5},
				{Time: time.Date(2019, time.January, 8, 0, 0, 0, 0, time.UTC), Value: 2},
				{Time: time.Date(2019, time.January, 9, 0, 0, 0, 0, time.UTC), Value: 3},
			},
			result: -0.75,
			start:  time.Date(2019, time.January, 5, 0, 0, 0, 0, time.UTC),
			end:    time.Date(2019, time.January, 8, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			wallet := PaperWallet{
				equityValues: tc.values,
			}

			max, start, end := wallet.MaxDrawdown()
			require.Equal(t, tc.result, max)
			require.Equal(t, tc.start, start)
			require.Equal(t, tc.end, end)
		})
	}
}

func TestPaperWallet_AssetsInfo(t *testing.T) {
	wallet := PaperWallet{}
	info, err := wallet.AssetsInfo("BTCUSDT")
	require.NoError(t, err)
	require.Equal(t, info.QuotePrecision, 8)
	require.Equal(t, info.GetBaseAsset(), "BTC")
	require.Equal(t, info.QuoteAsset, "USDT")
}

func TestPaperWallet_CreateOrderStop(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		wallet := NewPaperWallet(context.Background(), "USDT", getLog(), WithPaperAsset("USDT", 100))
		wallet.OnCandle(core.Candle{Pair: "BTCUSDT", Close: 100})
		_, err := wallet.CreateOrderMarket(context.Background(), core.SideTypeBuy, "BTCUSDT", 1)
		require.NoError(t, err)

		order, err := wallet.CreateOrderStop(context.Background(), "BTCUSDT", 1, 50)
		require.NoError(t, err)

		// create order and lock values
		require.Len(t, wallet.orders, 2)
		require.Equal(t, 1.0, order.Quantity)
		require.Equal(t, 50.0, order.Price)
		require.Equal(t, 50.0, *order.Stop)
		require.Equal(t, 0.0, wallet.assets["BTC"].Free)
		require.Equal(t, 1.0, wallet.assets["BTC"].Lock)

		// a new candle should execute order and unlock values
		wallet.OnCandle(core.Candle{Pair: "BTCUSDT", Close: 40})
		require.Equal(t, core.OrderStatusTypeFilled, wallet.orders[1].Status)
		require.Equal(t, 50.0, wallet.assets["USDT"].Free)
		require.Equal(t, 0.0, wallet.assets["USDT"].Lock)
		require.Equal(t, 0.0, wallet.assets["BTC"].Free)
		require.Equal(t, 0.0, wallet.assets["BTC"].Lock)
		require.Equal(t, 100.0, wallet.avgLongPrice["BTCUSDT"])
	})
}

func TestUpdateAveragePrice(t *testing.T) {
	t.Run("long", func(t *testing.T) {
		wallet := NewPaperWallet(
			context.Background(),
			"USDT",
			getLog(),
			WithPaperAsset("BTC", 0),
			WithPaperAsset("USDT", 100),
		)

		tt := []struct {
			name     string
			quantity float64
			price    float64
			avgPrice float64
		}{
			{
				name:     "first order",
				quantity: 1,
				price:    100,
				avgPrice: 100,
			},
			{
				name:     "second order",
				quantity: 1,
				price:    50,
				avgPrice: 75,
			},
			{
				name:     "third order",
				quantity: 2,
				price:    101,
				avgPrice: 88,
			},
		}

		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				wallet.updateAveragePrice(core.SideTypeBuy, "BTCUSDT", tc.quantity, tc.price)
				require.Equal(t, tc.avgPrice, wallet.avgLongPrice["BTCUSDT"])
				wallet.assets["BTC"].Free += tc.quantity
			})
		}
	})

	t.Run("short", func(t *testing.T) {
		wallet := NewPaperWallet(
			context.Background(),
			"USDT",
			getLog(),
			WithPaperAsset("BTC", 0),
			WithPaperAsset("USDT", 100),
		)

		tt := []struct {
			name     string
			quantity float64
			price    float64
			avgPrice float64
		}{
			{
				name:     "first order",
				quantity: 1,
				price:    100,
				avgPrice: 100,
			},
			{
				name:     "second order",
				quantity: 1,
				price:    50,
				avgPrice: 75,
			},
			{
				name:     "third order",
				quantity: 2,
				price:    101,
				avgPrice: 88,
			},
		}

		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				wallet.updateAveragePrice(core.SideTypeSell, "BTCUSDT", tc.quantity, tc.price)
				require.Equal(t, tc.avgPrice, wallet.avgShortPrice["BTCUSDT"])
				wallet.assets["BTC"].Free -= tc.quantity
			})
		}
	})

	t.Run("mixed order", func(t *testing.T) {
		wallet := NewPaperWallet(
			context.Background(),
			"USDT",
			getLog(),
			WithPaperAsset("BTC", 0),
			WithPaperAsset("USDT", 100),
		)

		tt := []struct {
			name          string
			side          core.SideType
			quantity      float64
			price         float64
			avgLongPrice  float64
			avgShortPrice float64
		}{
			{
				name:         "first buy order",
				side:         core.SideTypeBuy,
				quantity:     1,
				price:        100,
				avgLongPrice: 100,
			},
			{
				name:         "second buy order",
				side:         core.SideTypeBuy,
				quantity:     1,
				price:        50,
				avgLongPrice: 75,
			},
			{
				name:         "sell half",
				side:         core.SideTypeSell,
				quantity:     1,
				price:        50,
				avgLongPrice: 75,
			},
			{
				name:          "long to short",
				side:          core.SideTypeSell,
				quantity:      2,
				price:         100,
				avgLongPrice:  75,
				avgShortPrice: 100,
			},
			{
				name:          "back to long",
				side:          core.SideTypeBuy,
				quantity:      2,
				price:         50,
				avgLongPrice:  50,
				avgShortPrice: 100,
			},
		}

		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				wallet.updateAveragePrice(tc.side, "BTCUSDT", tc.quantity, tc.price)
				require.Equal(t, tc.avgLongPrice, wallet.avgLongPrice["BTCUSDT"])
				require.Equal(t, tc.avgShortPrice, wallet.avgShortPrice["BTCUSDT"])
				if tc.side == core.SideTypeBuy {
					wallet.assets["BTC"].Free += tc.quantity
				} else {
					wallet.assets["BTC"].Free -= tc.quantity
				}
			})
		}
	})

}
