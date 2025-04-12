package backnrun

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/aybabtme/uniplot/histogram"
	"github.com/raykavin/backnrun/internal/core"
	"github.com/raykavin/backnrun/internal/exchange"
	"github.com/raykavin/backnrun/internal/metric"
	"github.com/raykavin/backnrun/internal/notification"
	"github.com/raykavin/backnrun/internal/order"
	"github.com/raykavin/backnrun/internal/storage"
	"github.com/raykavin/backnrun/internal/strategy"
	"github.com/raykavin/backnrun/pkg/logger"
	"github.com/raykavin/backnrun/pkg/logger/zerolog"

	"github.com/olekukonko/tablewriter"
	"github.com/schollz/progressbar/v3"
)

const defaultDatabase = "backnrun.db"

type Backnrun struct {
	storage  core.OrderStorage
	settings core.Settings
	exchange core.Exchange
	strategy strategy.Strategy
	notifier core.Notifier
	telegram core.NotifierWithStart
	logger   logger.Logger

	orderController     *order.Controller
	priorityQueueCandle *core.PriorityQueue
	orderFeed           *order.Feed
	dataFeed            *exchange.DataFeedSubscription
	paperWallet         *exchange.PaperWallet

	strategiesControllers map[string]*strategy.Controller

	backtest bool
}

type Option func(*Backnrun)

// NewBot creates a new Backnrun bot instance with the provided settings and dependencies
func NewBot(ctx context.Context, settings core.Settings, exch core.Exchange, str strategy.Strategy,
	options ...Option) (*Backnrun, error) {

	// Initialize bot with required core components
	bot := &Backnrun{
		settings:              settings,
		exchange:              exch,
		strategy:              str,
		orderFeed:             order.NewOrderFeed(),
		dataFeed:              exchange.NewDataFeed(exch),
		strategiesControllers: make(map[string]*strategy.Controller),
		priorityQueueCandle:   core.NewPriorityQueue(nil),
	}

	// Validate trading pairs
	if err := validatePairs(settings.GetPairs()); err != nil {
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

	// Initialize logger
	if err := initializeLogger(bot); err != nil {
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

// initializeNotifications sets up notification systems like Telegram
func initializeNotifications(ctx context.Context, bot *Backnrun, settings core.Settings) error {
	var err error
	if settings.GetTelegram().IsEnabled() {
		bot.telegram, err = notification.NewTelegram(bot.orderController, settings)
		if err != nil {
			return err
		}
		// Register telegram as notifier
		WithNotifier(bot.telegram)(bot)
	}
	return nil
}

// initializeLogger sets up the logging system
func initializeLogger(bot *Backnrun) error {
	log, err := zerolog.NewZerolog("debug", "2006-01-02 15:04:05", true, false)
	if err != nil {
		return err
	}
	bot.logger = &zerolog.ZerologAdapter{Logger: log.Logger}
	return nil
}

// WithBacktest sets the bot to run in backtest mode, it is required for backtesting environments
// Backtest mode optimize the input read for CSV and deal with race conditions
func WithBacktest(wallet *exchange.PaperWallet) Option {
	return func(bot *Backnrun) {
		bot.backtest = true
		opt := WithPaperWallet(wallet)
		opt(bot)
	}
}

// WithStorage sets the storage for the bot, by default it uses a local file called ninjabot.db
func WithStorage(storage core.OrderStorage) Option {
	return func(bot *Backnrun) {
		bot.storage = storage
	}
}

// WithLogLevel sets the log level. eg: n.logger.DebugLevel, n.logger.InfoLevel, n.logger.WarnLevel, n.logger.ErrorLevel, n.logger.FatalLevel
func WithLogLevel(level logger.Level) Option {
	return func(n *Backnrun) {
		n.logger.SetLevel(level)
	}
}

// WithNotifier registers a notifier to the bot, currently only email and telegram are supported
func WithNotifier(notifier core.Notifier) Option {
	return func(bot *Backnrun) {
		bot.notifier = notifier
		bot.orderController.SetNotifier(notifier)
		bot.SubscribeOrder(notifier)
	}
}

// WithCandleSubscription subscribes a given struct to the candle feed
func WithCandleSubscription(subscriber core.CandleSubscriber) Option {
	return func(bot *Backnrun) {
		bot.SubscribeCandle(subscriber)
	}
}

// WithPaperWallet sets the paper wallet for the bot (used for backtesting and live simulation)
func WithPaperWallet(wallet *exchange.PaperWallet) Option {
	return func(bot *Backnrun) {
		bot.paperWallet = wallet
	}
}

func (n *Backnrun) SubscribeCandle(subscriptions ...core.CandleSubscriber) {
	for _, pair := range n.settings.GetPairs() {
		for _, subscription := range subscriptions {
			n.dataFeed.Subscribe(pair, n.strategy.Timeframe(), subscription.OnCandle, false)
		}
	}
}

func WithOrderSubscription(subscriber core.OrderSubscriber) Option {
	return func(bot *Backnrun) {
		bot.SubscribeOrder(subscriber)
	}
}

func (n *Backnrun) SubscribeOrder(subscriptions ...core.OrderSubscriber) {
	for _, pair := range n.settings.GetPairs() {
		for _, subscription := range subscriptions {
			n.orderFeed.Subscribe(pair, subscription.OnOrder, false)
		}
	}
}

func (n *Backnrun) Controller() *order.Controller {
	return n.orderController
}

// Summary function displays all trades, accuracy and some bot metric in stdout
// To access the raw data, you may access `bot.Controller().Results`
func (n *Backnrun) Summary() {
	var (
		total  float64
		wins   int
		loses  int
		volume float64
		sqn    float64
	)

	buffer := bytes.NewBuffer(nil)
	table := tablewriter.NewWriter(buffer)
	table.SetHeader([]string{"Pair", "Trades", "Win", "Loss", "% Win", "Payoff", "Pr Fact.", "SQN", "Profit", "Volume"})
	table.SetFooterAlignment(tablewriter.ALIGN_RIGHT)
	avgPayoff := 0.0
	avgProfitFactor := 0.0

	returns := make([]float64, 0)
	for _, summary := range n.orderController.Results {
		avgPayoff += summary.Payoff() * float64(len(summary.Win())+len(summary.Lose()))
		avgProfitFactor += summary.ProfitFactor() * float64(len(summary.Win())+len(summary.Lose()))
		table.Append([]string{
			summary.Pair,
			strconv.Itoa(len(summary.Win()) + len(summary.Lose())),
			strconv.Itoa(len(summary.Win())),
			strconv.Itoa(len(summary.Lose())),
			fmt.Sprintf("%.1f %%", float64(len(summary.Win()))/float64(len(summary.Win())+len(summary.Lose()))*100),
			fmt.Sprintf("%.3f", summary.Payoff()),
			fmt.Sprintf("%.3f", summary.ProfitFactor()),
			fmt.Sprintf("%.1f", summary.SQN()),
			fmt.Sprintf("%.2f", summary.Profit()),
			fmt.Sprintf("%.2f", summary.Volume),
		})
		total += summary.Profit()
		sqn += summary.SQN()
		wins += len(summary.Win())
		loses += len(summary.Lose())
		volume += summary.Volume

		returns = append(returns, summary.WinPercent()...)
		returns = append(returns, summary.LosePercent()...)
	}

	table.SetFooter([]string{
		"TOTAL",
		strconv.Itoa(wins + loses),
		strconv.Itoa(wins),
		strconv.Itoa(loses),
		fmt.Sprintf("%.1f %%", float64(wins)/float64(wins+loses)*100),
		fmt.Sprintf("%.3f", avgPayoff/float64(wins+loses)),
		fmt.Sprintf("%.3f", avgProfitFactor/float64(wins+loses)),
		fmt.Sprintf("%.1f", sqn/float64(len(n.orderController.Results))),
		fmt.Sprintf("%.2f", total),
		fmt.Sprintf("%.2f", volume),
	})
	table.Render()

	fmt.Println(buffer.String())
	fmt.Println("------ RETURN -------")
	totalReturn := 0.0
	returnsPercent := make([]float64, len(returns))
	for i, p := range returns {
		returnsPercent[i] = p * 100
		totalReturn += p
	}
	hist := histogram.Hist(15, returnsPercent)
	histogram.Fprint(os.Stdout, hist, histogram.Linear(10))
	fmt.Println()

	fmt.Println("------ CONFIDENCE INTERVAL (95%) -------")
	for pair, summary := range n.orderController.Results {
		fmt.Printf("| %s |\n", pair)
		returns := append(summary.WinPercent(), summary.LosePercent()...)
		returnsInterval := metric.Bootstrap(returns, metric.Mean, 10000, 0.95)
		payoffInterval := metric.Bootstrap(returns, metric.Payoff, 10000, 0.95)
		profitFactorInterval := metric.Bootstrap(returns, metric.ProfitFactor, 10000, 0.95)

		fmt.Printf("RETURN:      %.2f%% (%.2f%% ~ %.2f%%)\n",
			returnsInterval.Mean*100, returnsInterval.Lower*100, returnsInterval.Upper*100)
		fmt.Printf("PAYOFF:      %.2f (%.2f ~ %.2f)\n",
			payoffInterval.Mean, payoffInterval.Lower, payoffInterval.Upper)
		fmt.Printf("PROF.FACTOR: %.2f (%.2f ~ %.2f)\n",
			profitFactorInterval.Mean, profitFactorInterval.Lower, profitFactorInterval.Upper)
	}

	fmt.Println()

	if n.paperWallet != nil {
		n.paperWallet.Summary()
	}

}

func (n Backnrun) SaveReturns(outputDir string) error {
	for _, summary := range n.orderController.Results {
		outputFile := fmt.Sprintf("%s/%s.csv", outputDir, summary.Pair)
		if err := summary.SaveReturns(outputFile); err != nil {
			return err
		}
	}
	return nil
}

func (n *Backnrun) onCandle(candle core.Candle) {
	n.priorityQueueCandle.Push(candle)
}

func (n *Backnrun) processCandle(candle core.Candle) {
	if n.paperWallet != nil {
		n.paperWallet.OnCandle(candle)
	}

	n.strategiesControllers[candle.Pair].OnPartialCandle(candle)
	if candle.Complete {
		n.strategiesControllers[candle.Pair].OnCandle(candle)
		n.orderController.OnCandle(candle)
	}
}

// Process pending candles in buffer
func (n *Backnrun) processCandles() {
	for item := range n.priorityQueueCandle.PopLock() {
		n.processCandle(item.(core.Candle))
	}
}

// Start the backtest process and create a progress bar
// backtestCandles will process candles from a prirority queue in chronological order
func (n *Backnrun) backtestCandles() {
	n.logger.Info("[SETUP] Starting backtesting")

	progressBar := progressbar.Default(int64(n.priorityQueueCandle.Len()))
	for n.priorityQueueCandle.Len() > 0 {
		item := n.priorityQueueCandle.Pop()

		candle := item.(core.Candle)
		if n.paperWallet != nil {
			n.paperWallet.OnCandle(candle)
		}

		n.strategiesControllers[candle.Pair].OnPartialCandle(candle)
		if candle.Complete {
			n.strategiesControllers[candle.Pair].OnCandle(candle)
		}

		if err := progressBar.Add(1); err != nil {
			n.logger.Warnf("update progressbar fail: %v", err)
		}
	}
}

// Before Ninjabot start, we need to load the necessary data to fill strategy indicators
// Then, we need to get the time frame and warmup period to fetch the necessary candles
func (n *Backnrun) preload(ctx context.Context, pair string) error {
	if n.backtest {
		return nil
	}

	candles, err := n.exchange.CandlesByLimit(ctx, pair, n.strategy.Timeframe(), n.strategy.WarmupPeriod())
	if err != nil {
		return err
	}

	for _, candle := range candles {
		n.processCandle(candle)
	}

	n.dataFeed.Preload(pair, n.strategy.Timeframe(), candles)

	return nil
}

// Run will initialize the strategy controller, order controller, preload data and start the bot
func (n *Backnrun) Run(ctx context.Context) error {
	for _, pair := range n.settings.GetPairs() {
		// setup and subscribe strategy to data feed (candles)
		n.strategiesControllers[pair] = strategy.NewStrategyController(pair, n.strategy, n.orderController)

		// preload candles for warmup period
		err := n.preload(ctx, pair)
		if err != nil {
			return err
		}

		// link to ninja bot controller
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
