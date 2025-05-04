// Package exchange provides functionality for interacting with cryptocurrency exchanges
// and subscribing to data feeds for market information.
package exchange

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/StudioSol/set"
	"github.com/raykavin/backnrun/core"
)

// ---------------------
// Errors
// ---------------------

// Common errors that can occur during exchange operations
var (
	ErrInvalidQuantity   = errors.New("invalid quantity")
	ErrInsufficientFunds = errors.New("insufficient funds or locked")
	ErrInvalidAsset      = errors.New("invalid asset")
)

// ---------------------
// Types
// ---------------------

// DataFeedConsumer is a function type that processes candle data
type DataFeedConsumer func(core.Candle)

// Subscription represents a consumer subscription to a data feed
type Subscription struct {
	onCandleClose bool // Only process complete candles if true
	consumer      DataFeedConsumer
}

// DataFeed represents a data feed with channels for candles and errors
type DataFeed struct {
	Data chan core.Candle
	Err  chan error
}

// OrderError encapsulates an error related to an order
type OrderError struct {
	Err      error
	Pair     string
	Quantity float64
}

// Error implements the error interface
func (o *OrderError) Error() string {
	return fmt.Sprintf("order error: %v", o.Err)
}

// DataFeedSubscription manages subscriptions to data feeds
type DataFeedSubscription struct {
	exchange                core.Exchange
	feeds                   *set.LinkedHashSetString
	dataFeeds               map[string]*DataFeed
	subscriptionsByDataFeed map[string][]Subscription
	log                     core.Logger
	mu                      sync.RWMutex
}

// ---------------------
// Constructor
// ---------------------

// NewDataFeed creates a new instance of DataFeedSubscription
func NewDataFeed(exchange core.Exchange, log core.Logger) *DataFeedSubscription {
	return &DataFeedSubscription{
		exchange:                exchange,
		feeds:                   set.NewLinkedHashSetString(),
		log:                     log,
		dataFeeds:               make(map[string]*DataFeed),
		subscriptionsByDataFeed: make(map[string][]Subscription),
	}
}

// ---------------------
// Public Methods
// ---------------------

// Subscribe adds a new subscription for a pair and timeframe
func (d *DataFeedSubscription) Subscribe(pair, timeframe string, consumer DataFeedConsumer, onCandleClose bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	key := d.createFeedKey(pair, timeframe)
	d.feeds.Add(key)
	d.subscriptionsByDataFeed[key] = append(d.subscriptionsByDataFeed[key], Subscription{
		onCandleClose: onCandleClose,
		consumer:      consumer,
	})
}

// Preload loads historical candles for a specific subscription
func (d *DataFeedSubscription) Preload(ctx context.Context, pair, timeframe string, candles []core.Candle) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	d.log.Infof("preloading %d candles for %s-%s", len(candles), pair, timeframe)
	key := d.createFeedKey(pair, timeframe)

	// Send only complete candles
	for _, candle := range candles {
		if !candle.Complete {
			continue
		}

		for _, subscription := range d.subscriptionsByDataFeed[key] {
			subscription.consumer(candle)
		}
	}
}

// Connect establishes connections to the exchange and initializes feeds
func (d *DataFeedSubscription) Connect() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.log.Infof("Connecting to the exchange.")

	// Create a channel for each feed
	for feed := range d.feeds.Iter() {
		pair, timeframe := d.extractPairTimeframeFromKey(feed)
		candleChan, errChan := d.exchange.CandlesSubscription(context.Background(), pair, timeframe)
		d.dataFeeds[feed] = &DataFeed{
			Data: candleChan,
			Err:  errChan,
		}
	}
}

// Start begins processing all feeds
func (d *DataFeedSubscription) Start(ctx context.Context, waitForCompletion bool) {
	d.Connect()

	var wg sync.WaitGroup

	// Create a goroutine for each feed
	d.mu.RLock()
	for key, feed := range d.dataFeeds {
		wg.Add(1)
		go d.processFeed(ctx, key, feed, &wg)
	}
	d.mu.RUnlock()

	d.log.Infof("Data feed connected.")

	// Wait for completion if waitForCompletion is true
	if waitForCompletion {
		wg.Wait()
	}
}

// ---------------------
// Private Methods
// ---------------------

// processFeed processes candles received from a feed
func (d *DataFeedSubscription) processFeed(ctx context.Context, key string, feed *DataFeed, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return // Context cancelled, terminate the goroutine

		case candle, ok := <-feed.Data:
			if !ok {
				return // Channel closed, terminate goroutine
			}

			d.processCandle(key, candle)

		case err, ok := <-feed.Err:
			if !ok {
				return // Channel closed, terminate goroutine
			}

			if err != nil {
				d.log.Error("dataFeedSubscription/processFeed: ", err)
			}
		}
	}
}

// processCandle sends a candle to all subscribed consumers
func (d *DataFeedSubscription) processCandle(key string, candle core.Candle) {
	d.mu.RLock()
	subscriptions := d.subscriptionsByDataFeed[key]
	d.mu.RUnlock()

	// Send the candle to all subscribed consumers
	for _, subscription := range subscriptions {
		if subscription.onCandleClose && !candle.Complete {
			continue // Skip incomplete candles for onCandleClose subscriptions
		}
		subscription.consumer(candle)
	}
}

// ---------------------
// Helper Methods
// ---------------------

// createFeedKey generates a unique key for a pair and timeframe
func (d *DataFeedSubscription) createFeedKey(pair, timeframe string) string {
	return fmt.Sprintf("%s--%s", pair, timeframe)
}

// extractPairTimeframeFromKey extracts the pair and timeframe from a key
func (d *DataFeedSubscription) extractPairTimeframeFromKey(key string) (pair, timeframe string) {
	parts := strings.Split(key, "--")
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}
