package backnrun

import (
	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/exchange"
	"github.com/raykavin/backnrun/pkg/logger"
)

// Option is a functional option for configuring a Backnrun instance
type Option func(*Backnrun)

// WithBacktest sets the bot to run in backtest mode, it is required for backtesting environments
// Backtest mode optimize the input read for CSV and deal with race conditions
func WithBacktest(wallet *exchange.PaperWallet) Option {
	return func(bot *Backnrun) {
		bot.backtest = true
		opt := WithPaperWallet(wallet)
		opt(bot)
	}
}

// WithStorage sets the storage for the bot, by default it uses a local file called backnrun.db
func WithStorage(storage core.OrderStorage) Option {
	return func(bot *Backnrun) {
		bot.storage = storage
	}
}

// WithLogLevel sets the log level. eg: n.logger.DebugLevel, n.logger.InfoLevel, n.logger.WarnLevel, n.logger.ErrorLevel, n.logger.FatalLevel
func WithLogLevel(level logger.Level) Option {
	return func(n *Backnrun) {
		DefaultLog.SetLevel(level)
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

// WithOrderSubscription subscribes a given struct to the order feed
func WithOrderSubscription(subscriber core.OrderSubscriber) Option {
	return func(bot *Backnrun) {
		bot.SubscribeOrder(subscriber)
	}
}
