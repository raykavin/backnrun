package backnrun

import (
	"context"

	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/logger"
	"github.com/raykavin/backnrun/pkg/notification"
)

// initializeNotifications sets up notification systems like Telegram
func initializeNotifications(ctx context.Context, bot *Backnrun, settings *core.Settings, log logger.Logger) error {
	var err error
	if settings.Telegram.Enabled {
		bot.telegram, err = notification.NewTelegram(bot.orderController, settings, log)
		if err != nil {
			return err
		}
		// Register telegram as notifier
		WithNotifier(bot.telegram)(bot)
	}
	return nil
}

// SubscribeOrder subscribes the given subscribers to order updates for all pairs
func (n *Backnrun) SubscribeOrder(subscriptions ...core.OrderSubscriber) {
	for _, pair := range n.settings.Pairs {
		for _, subscription := range subscriptions {
			n.orderFeed.Subscribe(pair, subscription.OnOrder, false)
		}
	}
}

// SubscribeCandle subscribes the given subscribers to candle updates for all pairs
func (n *Backnrun) SubscribeCandle(subscriptions ...core.CandleSubscriber) {
	for _, pair := range n.settings.Pairs {
		for _, subscription := range subscriptions {
			n.dataFeed.Subscribe(pair, n.strategy.Timeframe(), subscription.OnCandle, false)
		}
	}
}
