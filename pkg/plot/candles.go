package plot

import (
	"time"

	"github.com/StudioSol/set"
	"github.com/raykavin/backnrun/pkg/core"
)

// OnCandle handles new candle events
func (c *Chart) OnCandle(candle core.Candle) {
	c.Lock()
	defer c.Unlock()

	// Initialize containers if needed
	if _, ok := c.candles[candle.Pair]; !ok {
		c.candles[candle.Pair] = make([]Candle, 0)
		c.ordersIDsByPair[candle.Pair] = set.NewLinkedHashSetINT64()
	}

	// Only add completed candles that are newer than our latest candle
	lastIndex := len(c.candles[candle.Pair]) - 1
	if candle.Complete && (len(c.candles[candle.Pair]) == 0 ||
		candle.Time.After(c.candles[candle.Pair][lastIndex].Time)) {

		// Add the candle to our collection
		c.candles[candle.Pair] = append(c.candles[candle.Pair], Candle{
			Time:   candle.Time,
			Open:   candle.Open,
			Close:  candle.Close,
			High:   candle.High,
			Low:    candle.Low,
			Volume: candle.Volume,
			Orders: make([]core.Order, 0),
		})

		// Initialize dataframe if needed
		if c.dataframe[candle.Pair] == nil {
			c.dataframe[candle.Pair] = &core.Dataframe{
				Pair:     candle.Pair,
				Metadata: make(map[string]core.Series[float64]),
			}
		}

		// Update dataframe with candle data
		df := c.dataframe[candle.Pair]
		df.Close = append(df.Close, candle.Close)
		df.Open = append(df.Open, candle.Open)
		df.High = append(df.High, candle.High)
		df.Low = append(df.Low, candle.Low)
		df.Volume = append(df.Volume, candle.Volume)
		df.Time = append(df.Time, candle.Time)
		df.LastUpdate = candle.Time

		// Copy metadata
		for k, v := range candle.Metadata {
			df.Metadata[k] = append(df.Metadata[k], v)
		}

		c.lastUpdate = time.Now()
	}
}

// candlesByPair returns candles with associated orders for a trading pair
func (c *Chart) candlesByPair(pair string) []Candle {
	if _, ok := c.candles[pair]; !ok {
		return []Candle{}
	}

	if _, ok := c.ordersIDsByPair[pair]; !ok {
		return c.candles[pair]
	}

	candles := make([]Candle, len(c.candles[pair]))
	copy(candles, c.candles[pair])

	// Track orders that have been assigned to candles
	orderCheck := make(map[int64]bool)
	for id := range c.ordersIDsByPair[pair].Iter() {
		orderCheck[id] = true
	}

	// Assign orders to the appropriate candle based on timestamp
	for i := range candles {
		for id := range c.ordersIDsByPair[pair].Iter() {
			order := c.orderByID[id]

			// Check if order timestamp falls within this candle's timeframe
			if (i < len(candles)-1 &&
				order.UpdatedAt.After(candles[i].Time) &&
				order.UpdatedAt.Before(candles[i+1].Time)) ||
				order.UpdatedAt.Equal(candles[i].Time) {

				delete(orderCheck, id)
				candles[i].Orders = append(candles[i].Orders, order)
			}
		}
	}

	// Assign remaining orders to the last candle if they occurred after it
	if len(candles) > 0 {
		lastCandle := &candles[len(candles)-1]

		for id := range orderCheck {
			order := c.orderByID[id]
			if order.UpdatedAt.After(lastCandle.Time) {
				lastCandle.Orders = append(lastCandle.Orders, order)
			}
		}
	}

	return candles
}
