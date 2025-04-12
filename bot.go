package backnrun

import (
	"context"
	"fmt"

	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/exchange"
	"github.com/raykavin/backnrun/pkg/logger"
	"github.com/raykavin/backnrun/pkg/order"
	"github.com/raykavin/backnrun/pkg/storage"
	"github.com/raykavin/backnrun/pkg/strategy"
)

// DefaultLog is the default logger instance
var DefaultLog logger.Logger

const defaultDatabase = "backnrun.db"

// Backnrun represents the main trading bot
type Backnrun struct {
	storage  core.OrderStorage
	exchange core.Exchange
	strategy core.Strategy
	notifier core.Notifier
	telegram core.NotifierWithStart

	orderFeed           *order.Feed
	settings            *core.Settings
	orderController     *order.Controller
	priorityQueueCandle *core.PriorityQueue
	dataFeed            *exchange.DataFeedSubscription
	paperWallet         *exchange.PaperWallet

	strategiesControllers map[string]*strategy.Controller

	backtest bool
}

// NewBot creates a new Backnrun bot instance with the provided settings and dependencies
func NewBot(
	ctx context.Context,
	settings *core.Settings,
	exch core.Exchange,
	strg core.Strategy,
	options ...Option,
) (*Backnrun, error) {

	// Initialize bot with required core components
	bot := &Backnrun{
		settings:              settings,
		exchange:              exch,
		strategy:              strg,
		orderFeed:             order.NewOrderFeed(),
		dataFeed:              exchange.NewDataFeed(exch, DefaultLog),
		strategiesControllers: make(map[string]*strategy.Controller),
		priorityQueueCandle:   core.NewPriorityQueue(nil),
	}

	// Validate trading pairs
	if err := validatePairs(settings.Pairs); err != nil {
		return nil, err
	}

	// Apply custom options
	for _, option := range options {
		option(bot)
	}

	// Initialize storage
	if err := initializeStorage(bot); err != nil {
		return nil, err
	}

	// Initialize order controller
	bot.orderController = order.NewController(ctx, exch, bot.storage, bot.orderFeed)

	// Initialize notification systems
	if err := initializeNotifications(ctx, bot, settings); err != nil {
		return nil, err
	}

	return bot, nil
}

// validatePairs ensures all trading pairs have valid asset and quote components
func validatePairs(pairs []string) error {
	for _, pair := range pairs {
		asset, quote := exchange.SplitAssetQuote(pair)
		if asset == "" || quote == "" {
			return fmt.Errorf("invalid pair: %s", pair)
		}
	}
	return nil
}

// initializeStorage sets up the bot's data storage
func initializeStorage(bot *Backnrun) error {
	var err error
	if bot.storage == nil {
		bot.storage, err = storage.FromFile(defaultDatabase)
		if err != nil {
			return err
		}
	}
	return nil
}

// Controller returns the order controller
func (n *Backnrun) Controller() *order.Controller {
	return n.orderController
}

// Run will initialize the strategy controller, order controller, preload data and start the bot
func (n *Backnrun) Run(ctx context.Context) error {
	for _, pair := range n.settings.Pairs {
		// setup and subscribe strategy to data feed (candles)
		n.strategiesControllers[pair] = strategy.NewStrategyController(pair, n.strategy, n.orderController, DefaultLog)

		// preload candles for warmup period
		err := n.preload(ctx, pair)
		if err != nil {
			return err
		}

		// link to backnrun controller
		n.dataFeed.Subscribe(pair, n.strategy.Timeframe(), n.onCandle, false)

		// start strategy controller
		n.strategiesControllers[pair].Start()
	}

	// start order feed and controller
	n.orderFeed.Start()
	n.orderController.Start()
	defer n.orderController.Stop()
	if n.telegram != nil {
		n.telegram.Start()
	}

	// start data feed and receives new candles
	n.dataFeed.Start(n.backtest)

	// start processing new candles for production or backtesting environment
	if n.backtest {
		n.backtestCandles()
	} else {
		n.processCandles()
	}

	return nil
}
