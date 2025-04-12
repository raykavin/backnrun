package exchange

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/StudioSol/set"
	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/logger"
)

// Erros comuns
var (
	ErrInvalidQuantity   = errors.New("invalid quantity")
	ErrInsufficientFunds = errors.New("insufficient funds or locked")
	ErrInvalidAsset      = errors.New("invalid asset")
)

// DataFeed representa um feed de dados com canais para candles e erros
type DataFeed struct {
	Data chan core.Candle
	Err  chan error
}

// DataFeedSubscription gerencia assinaturas de feeds de dados
type DataFeedSubscription struct {
	exchange                core.Exchange
	Feeds                   *set.LinkedHashSetString
	DataFeeds               map[string]*DataFeed
	SubscriptionsByDataFeed map[string][]Subscription
	log                     logger.Logger
	mu                      sync.RWMutex
}

// Subscription representa uma assinatura para um feed de dados
type Subscription struct {
	onCandleClose bool
	consumer      DataFeedConsumer
}

// OrderError encapsula um erro relacionado a uma ordem
type OrderError struct {
	Err      error
	Pair     string
	Quantity float64
}

// Error implementa a interface error
func (o *OrderError) Error() string {
	return fmt.Sprintf("order error: %v", o.Err)
}

// DataFeedConsumer é uma função que consome candles
type DataFeedConsumer func(core.Candle)

// NewDataFeed cria uma nova instância de DataFeedSubscription
func NewDataFeed(exchange core.Exchange, logger logger.Logger) *DataFeedSubscription {
	return &DataFeedSubscription{
		exchange:                exchange,
		Feeds:                   set.NewLinkedHashSetString(),
		log:                     logger,
		DataFeeds:               make(map[string]*DataFeed),
		SubscriptionsByDataFeed: make(map[string][]Subscription),
	}
}

// feedKey gera uma chave única para um par e timeframe
func (d *DataFeedSubscription) feedKey(pair, timeframe string) string {
	return fmt.Sprintf("%s--%s", pair, timeframe)
}

// pairTimeframeFromKey extrai o par e timeframe de uma chave
func (d *DataFeedSubscription) pairTimeframeFromKey(key string) (pair, timeframe string) {
	parts := strings.Split(key, "--")
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

// Subscribe adiciona uma nova assinatura para um par e timeframe
func (d *DataFeedSubscription) Subscribe(pair, timeframe string, consumer DataFeedConsumer, onCandleClose bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	key := d.feedKey(pair, timeframe)
	d.Feeds.Add(key)
	d.SubscriptionsByDataFeed[key] = append(d.SubscriptionsByDataFeed[key], Subscription{
		onCandleClose: onCandleClose,
		consumer:      consumer,
	})
}

// Preload carrega candles históricos para todas as assinaturas
func (d *DataFeedSubscription) Preload(pair, timeframe string, candles []core.Candle) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	d.log.Infof("preloading %d candles for %s-%s", len(candles), pair, timeframe)
	key := d.feedKey(pair, timeframe)

	// Envia apenas candles completos
	for _, candle := range candles {
		if !candle.Complete {
			continue
		}

		for _, subscription := range d.SubscriptionsByDataFeed[key] {
			subscription.consumer(candle)
		}
	}
}

// Connect estabelece conexão com o exchange e inicia os feeds
func (d *DataFeedSubscription) Connect() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.log.Infof("Connecting to the exchange.")

	// Cria um canal para cada feed
	for feed := range d.Feeds.Iter() {
		pair, timeframe := d.pairTimeframeFromKey(feed)
		ccandle, cerr := d.exchange.CandlesSubscription(context.Background(), pair, timeframe)
		d.DataFeeds[feed] = &DataFeed{
			Data: ccandle,
			Err:  cerr,
		}
	}
}

// Start inicia o processamento de todos os feeds
func (d *DataFeedSubscription) Start(loadSync bool) {
	d.Connect()

	var wg sync.WaitGroup

	// Cria uma goroutine para cada feed
	d.mu.RLock()
	for key, feed := range d.DataFeeds {
		wg.Add(1)
		go d.processFeed(key, feed, &wg)
	}
	d.mu.RUnlock()

	d.log.Infof("Data feed connected.")

	// Espera a conclusão se loadSync for true
	if loadSync {
		wg.Wait()
	}
}

// processFeed processa os candles recebidos de um feed
func (d *DataFeedSubscription) processFeed(key string, feed *DataFeed, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case candle, ok := <-feed.Data:
			if !ok {
				return // Canal fechado, encerra a goroutine
			}

			d.mu.RLock()
			subscriptions := d.SubscriptionsByDataFeed[key]
			d.mu.RUnlock()

			// Envia o candle para todos os consumidores inscritos
			for _, subscription := range subscriptions {
				if subscription.onCandleClose && !candle.Complete {
					continue
				}
				subscription.consumer(candle)
			}

		case err, ok := <-feed.Err:
			if !ok {
				return // Canal fechado, encerra a goroutine
			}

			if err != nil {
				d.log.Error("dataFeedSubscription/processFeed: ", err)
			}
		}
	}
}
