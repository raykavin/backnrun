package bot

import (
	"context"
	"fmt"

	"github.com/raykavin/backnrun/core"
	"github.com/raykavin/backnrun/exchange"
	"github.com/raykavin/backnrun/order"
	"github.com/raykavin/backnrun/storage"
	strg "github.com/raykavin/backnrun/strategy"
)

const defaultDatabase = "bot.db"

// Bot represents the main trading bot
type Bot struct {
	storage  core.Storage
	exchange core.Exchange
	strategy core.Strategy
	notifier core.Notifier
	telegram core.NotifierWithStart
	log      core.Logger

	orderFeed           *order.Feed
	settings            *core.Settings
	orderController     *order.Controller
	priorityQueueCandle *core.PriorityQueue
	dataFeed            *exchange.DataFeedSubscription
	paperWallet         *exchange.PaperWallet

	strategiesControllers map[string]*strg.Controller

	backtest bool
}

// NewBot creates a new Bot bot instance with the provided settings and dependencies
func NewBot(
	ctx context.Context,
	settings *core.Settings,
	exch core.Exchange,
	strategy core.Strategy,
	log core.Logger,
	options ...Option,
) (*Bot, error) {
	// Validate parameters
	err := validate(settings, exch, strategy, log)
	if err != nil {
		return nil, err
	}

	// Initialize bot with required core components
	bot := &Bot{
		settings:              settings,
		exchange:              exch,
		strategy:              strategy,
		orderFeed:             order.NewOrderFeed(),
		dataFeed:              exchange.NewDataFeed(exch, log),
		log:                   log,
		priorityQueueCandle:   core.NewPriorityQueue(nil),
		strategiesControllers: make(map[string]*strg.Controller),
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
	bot.orderController = order.NewController(ctx, exch, bot.storage, log, bot.orderFeed)

	// Initialize notification systems
	if err := initializeNotifications(ctx, bot, settings, log); err != nil {
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

// validate checks if the provided settings, exchange, strategy, and logger are valid
func validate(settings *core.Settings, exch core.Exchange, strategy core.Strategy, log core.Logger) error {
	if settings == nil || len(settings.Pairs) == 0 {
		return fmt.Errorf("settings cannot be nil")
	}

	if exch == nil {
		return fmt.Errorf("exchange cannot be nil")
	}

	if strategy == nil {
		return fmt.Errorf("strategy cannot be nil")
	}

	if log == nil {
		return fmt.Errorf("logger cannot be nil")
	}

	return nil
}

// initializeStorage sets up the bot's data storage
func initializeStorage(bot *Bot) error {
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
func (n *Bot) Controller() *order.Controller {
	return n.orderController
}

// Run will initialize the strategy controller, order controller, preload data and start the bot
func (n *Bot) Run(ctx context.Context) error {
	for _, pair := range n.settings.Pairs {
		// setup and subscribe strategy to data feed (candles)
		n.strategiesControllers[pair] = strg.NewStrategyController(pair, n.strategy, n.orderController, n.log)

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
	n.orderController.Start(ctx)
	defer n.orderController.Stop(ctx)
	if n.telegram != nil {
		n.telegram.Start()
	}

	// start data feed and receives new candles
	n.dataFeed.Start(ctx, n.backtest)

	// start processing new candles for production or backtesting environment
	if n.backtest {
		n.backtestCandles(ctx)
	} else {
		n.processCandles(ctx)
	}

	return nil
}
