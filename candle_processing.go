package backnrun

import (
	"context"
	"time"

	"github.com/raykavin/backnrun/pkg/core"
	"github.com/schollz/progressbar/v3"
)

// onCandle handles incoming candles and adds them to the priority queue
func (n *Backnrun) onCandle(candle core.Candle) {
	n.priorityQueueCandle.Push(candle)
}

// processCandle processes a single candle through the bot's systems
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

// processCandles processes pending candles in buffer
func (n *Backnrun) processCandles() {
	for item := range n.priorityQueueCandle.PopLock() {
		n.processCandle(item.(core.Candle))
	}
}

// backtestCandles processes candles for backtesting with a progress bar
func (n *Backnrun) backtestCandles() {
	n.log.Info("Starting backtesting")

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
			n.log.Warnf("update progressbar fail: %v", err)
		}

		time.Sleep(5 * time.Millisecond) // prevent CPU overload
	}
}

// preload loads initial data needed for strategy indicators
func (n *Backnrun) preload(ctx context.Context, pair string) error {
	if n.backtest {
		return nil
	}

	candles, err := n.exchange.CandlesByLimit(
		ctx,
		pair,
		n.strategy.Timeframe(),
		n.strategy.WarmupPeriod(),
	)

	if err != nil {
		return err
	}

	for _, candle := range candles {
		n.processCandle(candle)
	}

	n.dataFeed.Preload(pair, n.strategy.Timeframe(), candles)

	return nil
}
