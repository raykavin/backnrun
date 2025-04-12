package order

import (
	"sync"

	"github.com/raykavin/backnrun/pkg/core"
)

// FeedConsumer is a function type that processes order events
type FeedConsumer func(order core.Order)

// DataFeed represents channels for order data and errors
type DataFeed struct {
	Data chan core.Order
	Err  chan error
}

// Subscription represents a consumer subscription to order updates
type Subscription struct {
	onlyNewOrder bool
	consumer     FeedConsumer
}

// Feed manages order data feeds and subscriptions
type Feed struct {
	mu                    sync.RWMutex
	OrderFeeds            map[string]*DataFeed
	SubscriptionsBySymbol map[string][]Subscription
}

// NewOrderFeed creates a new order feed manager
func NewOrderFeed() *Feed {
	return &Feed{
		OrderFeeds:            make(map[string]*DataFeed),
		SubscriptionsBySymbol: make(map[string][]Subscription),
	}
}

// Subscribe registers a consumer to receive order updates for a specific pair
func (f *Feed) Subscribe(pair string, consumer FeedConsumer, onlyNewOrder bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Create a new data feed if one doesn't exist for this pair
	if _, ok := f.OrderFeeds[pair]; !ok {
		f.OrderFeeds[pair] = &DataFeed{
			Data: make(chan core.Order, 100), // Buffered channel to prevent blocking
			Err:  make(chan error, 100),
		}
	}

	// Add the subscription
	f.SubscriptionsBySymbol[pair] = append(f.SubscriptionsBySymbol[pair], Subscription{
		onlyNewOrder: onlyNewOrder,
		consumer:     consumer,
	})
}

// Publish sends an order update to all subscribers for the pair
// Note: The isNew parameter is currently unused but kept for API compatibility
func (f *Feed) Publish(order core.Order, isNew bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if feed, ok := f.OrderFeeds[order.Pair]; ok {
		// Non-blocking send - drop updates if no one is listening
		select {
		case feed.Data <- order:
			// Successfully sent
		default:
			// Channel full, could log this situation
		}
	}
}

// Start begins processing order updates for all registered feeds
func (f *Feed) Start() {
	f.mu.RLock()
	defer f.mu.RUnlock()

	for pair, feed := range f.OrderFeeds {
		go f.processOrdersForPair(pair, feed)
	}
}

// processOrdersForPair handles order updates for a specific trading pair
func (f *Feed) processOrdersForPair(pair string, feed *DataFeed) {
	for order := range feed.Data {
		f.mu.RLock()
		subscriptions := f.SubscriptionsBySymbol[pair]
		f.mu.RUnlock()

		// Distribute the order to all subscribers
		for _, subscription := range subscriptions {
			// Pass the order to the consumer
			// Note: In a future improvement, we could respect the onlyNewOrder flag
			subscription.consumer(order)
		}
	}
}

// Stop gracefully shuts down all feed channels
func (f *Feed) Stop() {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Close all channels
	for pair, feed := range f.OrderFeeds {
		close(feed.Data)
		close(feed.Err)
		delete(f.OrderFeeds, pair)
	}

	// Clear subscriptions
	f.SubscriptionsBySymbol = make(map[string][]Subscription)
}
