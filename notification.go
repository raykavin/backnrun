package backnrun

import (
	"context"

	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/logger"
	"github.com/raykavin/backnrun/pkg/notification"
)

// initializeNotifications sets up notification systems like Telegram
func initializeNotifications(ctx context.Context, bot *Bot, settings *core.Settings, log logger.Logger) error {
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
func (bot *Bot) SubscribeOrder(subscriptions ...core.OrderSubscriber) {
	for _, pair := range bot.settings.Pairs {
		for _, subscription := range subscriptions {
			bot.orderFeed.Subscribe(pair, subscription.OnOrder, false)
		}
	}
}

// SubscribeCandle subscribes the given subscribers to candle updates for all pairs
func (bot *Bot) SubscribeCandle(subscriptions ...core.CandleSubscriber) {
	for _, pair := range bot.settings.Pairs {
		for _, subscription := range subscriptions {
			bot.dataFeed.Subscribe(pair, bot.strategy.Timeframe(), subscription.OnCandle, false)
		}
	}
}
