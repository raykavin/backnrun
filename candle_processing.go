package backnrun

import (
	"context"
	"time"

	"github.com/raykavin/backnrun/pkg/core"
)

// onCandle handles incoming candles and adds them to the priority queue
func (bot *Bot) onCandle(candle core.Candle) {
	bot.priorityQueueCandle.Push(candle)
}

// processCandle processes a single candle through the bot's systems
func (bot *Bot) processCandle(candle core.Candle) {
	if bot.paperWallet != nil {
		bot.paperWallet.OnCandle(candle)
	}

	bot.strategiesControllers[candle.Pair].OnPartialCandle(candle)
	if candle.Complete {
		bot.strategiesControllers[candle.Pair].OnCandle(candle)
		bot.orderController.OnCandle(candle)
	}
}

// processCandles processes pending candles in buffer
func (bot *Bot) processCandles() {
	for item := range bot.priorityQueueCandle.PopLock() {
		bot.processCandle(item.(core.Candle))
	}
}

// backtestCandles processes candles for backtesting with a progress bar
func (bot *Bot) backtestCandles() {
	bot.log.Info("Starting backtesting...")

	for bot.priorityQueueCandle.Len() > 0 {
		item := bot.priorityQueueCandle.Pop()

		candle := item.(core.Candle)
		if bot.paperWallet != nil {
			bot.paperWallet.OnCandle(candle)
		}

		bot.strategiesControllers[candle.Pair].OnPartialCandle(candle)
		if candle.Complete {
			bot.strategiesControllers[candle.Pair].OnCandle(candle)
		}

		time.Sleep(5 * time.Millisecond) // prevent CPU overload
	}
}

// preload loads initial data needed for strategy indicators
func (bot *Bot) preload(ctx context.Context, pair string) error {
	if bot.backtest {
		return nil
	}

	candles, err := bot.exchange.CandlesByLimit(
		ctx,
		pair,
		bot.strategy.Timeframe(),
		bot.strategy.WarmupPeriod(),
	)

	if err != nil {
		return err
	}

	for _, candle := range candles {
		bot.processCandle(candle)
	}

	bot.dataFeed.Preload(pair, bot.strategy.Timeframe(), candles)

	return nil
}
