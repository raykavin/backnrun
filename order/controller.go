package order

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/raykavin/backnrun/core"
	"github.com/raykavin/backnrun/exchange"
)

// Status represents the current state of the order controller
type Status string

// Available controller statuses
const (
	StatusRunning Status = "running"
	StatusStopped Status = "stopped"
	StatusError   Status = "error"
)

// Controller manages orders, positions, and trading operations
type Controller struct {
	ctx            context.Context
	exchange       core.Exchange
	storage        core.Storage
	log            core.Logger
	mu             sync.Mutex
	orderFeed      *Feed
	notifier       core.Notifier
	Results        map[string]*TradeSummary
	lastPrice      map[string]float64
	tickerInterval time.Duration
	finish         chan bool
	status         Status
	position       map[string]*Position
}

// NewController creates a new order controller
func NewController(
	ctx context.Context,
	exchange core.Exchange,
	storage core.Storage,
	log core.Logger,
	orderFeed *Feed,
) *Controller {

	return &Controller{
		ctx:            ctx,
		storage:        storage,
		exchange:       exchange,
		orderFeed:      orderFeed,
		tickerInterval: time.Second,
		log:            log,
		lastPrice:      make(map[string]float64),
		Results:        make(map[string]*TradeSummary),
		finish:         make(chan bool),
		position:       make(map[string]*Position),
	}
}

// SetNotifier configures a notifier core for the controller
func (c *Controller) SetNotifier(notifier core.Notifier) {
	c.notifier = notifier
}

// OnCandle updates the last known price for a trading pair
func (c *Controller) OnCandle(candle core.Candle) {
	c.lastPrice[candle.Pair] = candle.Close
}

// Status returns the current controller status
func (c *Controller) Status() Status {
	return c.status
}

// Start begins the order monitoring process
func (c *Controller) Start(ctx context.Context) {
	if c.status != StatusRunning {
		c.status = StatusRunning
		go func() {
			ticker := time.NewTicker(c.tickerInterval)
			for {
				select {
				case <-ticker.C:
					c.updateOrders(ctx)
				case <-c.finish:
					ticker.Stop()
					return
				}
			}
		}()

		c.log.Info("Bot started.")
	}
}

// Stop halts the order monitoring process
func (c *Controller) Stop(ctx context.Context) {
	if c.status == StatusRunning {
		c.status = StatusStopped
		c.updateOrders(ctx)
		c.finish <- true
		c.log.Info("Bot stopped")
	}
}

// Account retrieves the current trading account information
func (c *Controller) Account(ctx context.Context) (core.Account, error) {
	return c.exchange.Account(ctx)
}

// Position retrieves the current asset and quote balances for a trading pair
func (c *Controller) Position(ctx context.Context, pair string) (asset, quote float64, err error) {
	return c.exchange.Position(ctx, pair)
}

// LastQuote retrieves the most recent price for a trading pair
func (c *Controller) LastQuote(pair string) (float64, error) {
	return c.exchange.LastQuote(c.ctx, pair)
}

// PositionValue calculates the current value of holdings for a trading pair
func (c *Controller) PositionValue(ctx context.Context, pair string) (float64, error) {
	asset, _, err := c.exchange.Position(ctx, pair)
	if err != nil {
		return 0, err
	}
	return asset * c.lastPrice[pair], nil
}

// Order retrieves information about a specific order
func (c *Controller) Order(ctx context.Context, pair string, id int64) (core.Order, error) {
	return c.exchange.Order(ctx, pair, id)
}

// CreateOrderOCO creates a One-Cancels-the-Other order pair
func (c *Controller) CreateOrderOCO(ctx context.Context, side core.SideType, pair string, size, price, stop,
	stopLimit float64) ([]core.Order, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.log.Infof("Creating OCO order for %s", pair)
	orders, err := c.exchange.CreateOrderOCO(ctx, side, pair, size, price, stop, stopLimit)
	if err != nil {
		c.notifyError(err)
		return nil, err
	}

	for i := range orders {
		err := c.storage.CreateOrder(ctx, &orders[i])
		if err != nil {
			c.notifyError(err)
			return nil, err
		}
		go c.orderFeed.Publish(orders[i], true)
	}

	return orders, nil
}

// CreateOrderLimit creates a limit order
func (c *Controller) CreateOrderLimit(ctx context.Context, side core.SideType, pair string, size, limit float64) (core.Order, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.log.Infof("Creating LIMIT %s order for %s", side, pair)
	order, err := c.exchange.CreateOrderLimit(ctx, side, pair, size, limit)
	if err != nil {
		c.notifyError(err)
		return core.Order{}, err
	}

	err = c.storage.CreateOrder(ctx, &order)
	if err != nil {
		c.notifyError(err)
		return core.Order{}, err
	}
	go c.orderFeed.Publish(order, true)
	c.log.Infof("[ORDER CREATED] %s", order)
	return order, nil
}

// CreateOrderMarketQuote creates a market order with a specified quote amount
func (c *Controller) CreateOrderMarketQuote(ctx context.Context, side core.SideType, pair string, amount float64) (core.Order, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.log.Infof("Creating MARKET %s order for %s", side, pair)
	order, err := c.exchange.CreateOrderMarketQuote(ctx, side, pair, amount)
	if err != nil {
		c.notifyError(err)
		return core.Order{}, err
	}

	err = c.storage.CreateOrder(ctx, &order)
	if err != nil {
		c.notifyError(err)
		return core.Order{}, err
	}

	// calculate profit
	c.processTrade(&order)
	go c.orderFeed.Publish(order, true)
	c.log.Infof("[ORDER CREATED] %s", order)
	return order, err
}

// CreateOrderMarket creates a market order with a specified size
func (c *Controller) CreateOrderMarket(ctx context.Context, side core.SideType, pair string, size float64) (core.Order, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.log.Infof("Creating MARKET %s order for %s", side, pair)
	order, err := c.exchange.CreateOrderMarket(ctx, side, pair, size)
	if err != nil {
		c.notifyError(err)
		return core.Order{}, err
	}

	err = c.storage.CreateOrder(ctx, &order)
	if err != nil {
		c.notifyError(err)
		return core.Order{}, err
	}

	// calculate profit
	c.processTrade(&order)
	go c.orderFeed.Publish(order, true)
	c.log.Infof("[ORDER CREATED] %s", order)
	return order, err
}

// CreateOrderStop creates a stop loss order
func (c *Controller) CreateOrderStop(ctx context.Context, pair string, size float64, limit float64) (core.Order, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.log.Infof("Creating STOP order for %s", pair)
	order, err := c.exchange.CreateOrderStop(ctx, pair, size, limit)
	if err != nil {
		c.notifyError(err)
		return core.Order{}, err
	}

	err = c.storage.CreateOrder(ctx, &order)
	if err != nil {
		c.notifyError(err)
		return core.Order{}, err
	}
	go c.orderFeed.Publish(order, true)
	c.log.Infof("[ORDER CREATED] %s", order)
	return order, nil
}

// Cancel cancels an existing order
func (c *Controller) Cancel(ctx context.Context, order core.Order) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.log.Infof("Cancelling order for %s", order.Pair)
	err := c.exchange.Cancel(ctx, order)
	if err != nil {
		return err
	}

	order.Status = core.OrderStatusTypePendingCancel
	err = c.storage.UpdateOrder(ctx, &order)
	if err != nil {
		c.notifyError(err)
		return err
	}
	c.log.Infof("[ORDER CANCELED] %s", order)
	return nil
}

// updateOrders checks for status changes in pending orders
func (c *Controller) updateOrders(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Get pending orders
	orders, err := c.storage.Orders(ctx, core.WithStatusIn(
		core.OrderStatusTypeNew,
		core.OrderStatusTypePartiallyFilled,
		core.OrderStatusTypePendingCancel,
	))
	if err != nil {
		c.notifyError(err)
		return
	}

	// For each pending order, check for updates
	var updatedOrders []core.Order
	for _, order := range orders {
		excOrder, err := c.exchange.Order(ctx, order.Pair, order.ExchangeID)
		if err != nil {
			c.log.WithField("id", order.ExchangeID).Error("orderController/get: ", err)
			continue
		}

		// No status change
		if excOrder.Status == order.Status {
			continue
		}

		excOrder.ID = order.ID
		err = c.storage.UpdateOrder(ctx, &excOrder)
		if err != nil {
			c.notifyError(err)
			continue
		}

		c.log.Infof("[ORDER %s] %s", excOrder.Status, excOrder)
		updatedOrders = append(updatedOrders, excOrder)
	}

	for _, processOrder := range updatedOrders {
		c.processTrade(&processOrder)
		c.orderFeed.Publish(processOrder, false)
	}
}

// processTrade updates the trade summary and position data when an order is filled
func (c *Controller) processTrade(order *core.Order) {
	if order.Status != core.OrderStatusTypeFilled {
		return
	}

	// Initialize results map if needed
	if _, ok := c.Results[order.Pair]; !ok {
		c.Results[order.Pair] = &TradeSummary{Pair: order.Pair}
	}

	// Register order volume
	c.Results[order.Pair].Volume += order.Price * order.Quantity

	// Update position size / avg price
	c.updatePosition(order)
}

// updatePosition updates the current position based on a new order
func (c *Controller) updatePosition(o *core.Order) {
	// Get filled orders before the current order
	position, ok := c.position[o.Pair]
	if !ok {
		c.position[o.Pair] = &Position{
			AvgPrice:  o.Price,
			Quantity:  o.Quantity,
			CreatedAt: o.CreatedAt,
			Side:      o.Side,
		}
		return
	}

	result, closed := position.Update(o)
	if closed {
		delete(c.position, o.Pair)
	}

	if result != nil {
		c.recordTradeResult(o.Pair, result)
		c.notifyTradeResult(o.Pair, result)
	}
}

// recordTradeResult updates the trade summary with a new trade result
func (c *Controller) recordTradeResult(pair string, result *TradeResult) {
	summary := c.Results[pair]

	if result.ProfitPercent >= 0 {
		if result.Side == core.SideTypeBuy {
			summary.WinLong = append(summary.WinLong, result.ProfitValue)
			summary.WinLongPercent = append(summary.WinLongPercent, result.ProfitPercent)
		} else {
			summary.WinShort = append(summary.WinShort, result.ProfitValue)
			summary.WinShortPercent = append(summary.WinShortPercent, result.ProfitPercent)
		}
	} else {
		if result.Side == core.SideTypeBuy {
			summary.LoseLong = append(summary.LoseLong, result.ProfitValue)
			summary.LoseLongPercent = append(summary.LoseLongPercent, result.ProfitPercent)
		} else {
			summary.LoseShort = append(summary.LoseShort, result.ProfitValue)
			summary.LoseShortPercent = append(summary.LoseShortPercent, result.ProfitPercent)
		}
	}
}

// notifyTradeResult sends a notification about a completed trade
func (c *Controller) notifyTradeResult(pair string, result *TradeResult) {
	_, quote := exchange.SplitAssetQuote(pair)

	c.notify(fmt.Sprintf("[PROFIT] %f %s (%f %%)\n",
		result.ProfitValue, quote, result.ProfitPercent*100), true)

	c.notify(c.Results[pair].String())
}

// notify sends a message through the logging system and notifier
func (c *Controller) notify(message string, withLogger ...bool) {
	if len(withLogger) > 0 && withLogger[0] {
		c.log.Info(message)
	} else {
		fmt.Println(message)
	}

	if c.notifier != nil {
		c.notifier.Notify(message)
	}
}

// notifyError sends an error through the logging system and notifier
func (c *Controller) notifyError(err error) {
	c.log.Error(err)
	if c.notifier != nil {
		c.notifier.OnError(err)
	}
}
