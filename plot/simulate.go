package plot

import (
	"math/rand"
	"time"

	"github.com/raykavin/backnrun/core"
)

// StartCandleSimulation starts a simulation that generates candle updates at regular intervals
// This is useful for testing real-time updates
func (c *Chart) StartCandleSimulation(pair string, interval time.Duration) {
	// Create a ticker that fires at the specified interval
	ticker := time.NewTicker(interval)

	// Start a goroutine to generate candle updates
	go func() {
		// Get the last candle for the pair
		c.Lock()
		var lastCandle Candle
		if len(c.candles[pair]) > 0 {
			lastCandle = c.candles[pair][len(c.candles[pair])-1]
		} else {
			// If no candles exist, create a default one
			lastCandle = Candle{
				Time:   time.Now(),
				Open:   100.0,
				High:   105.0,
				Low:    95.0,
				Close:  100.0,
				Volume: 1000.0,
			}
		}
		c.Unlock()

		// Keep track of the current candle
		currentCandle := core.Candle{
			Pair:     pair,
			Time:     lastCandle.Time,
			Open:     lastCandle.Open,
			High:     lastCandle.High,
			Low:      lastCandle.Low,
			Close:    lastCandle.Close,
			Volume:   lastCandle.Volume,
			Complete: false,
		}

		// Generate updates at regular intervals
		for range ticker.C {
			// Update the current candle with a random price movement
			priceChange := (rand.Float64() - 0.5) * 2.0 // Random value between -1.0 and 1.0
			newClose := currentCandle.Close + priceChange

			// Update high and low if needed
			if newClose > currentCandle.High {
				currentCandle.High = newClose
			}
			if newClose < currentCandle.Low {
				currentCandle.Low = newClose
			}

			// Update close price and volume
			currentCandle.Close = newClose
			currentCandle.Volume += rand.Float64() * 10.0

			// Process the updated candle
			c.OnCandle(currentCandle)

			// Every 10 updates, complete the current candle and start a new one
			if rand.Intn(10) == 0 {
				// Complete the current candle
				currentCandle.Complete = true
				c.OnCandle(currentCandle)

				// Start a new candle
				currentCandle = core.Candle{
					Pair:     pair,
					Time:     currentCandle.Time.Add(time.Minute), // Move forward by 1 minute
					Open:     currentCandle.Close,                 // Open at the previous close
					High:     currentCandle.Close,
					Low:      currentCandle.Close,
					Close:    currentCandle.Close,
					Volume:   0.0,
					Complete: false,
				}
			}
		}
	}()
}

// StopCandleSimulation stops the candle simulation
func (c *Chart) StopCandleSimulation() {
	// This is a placeholder for stopping the simulation
	// In a real implementation, you would need to keep track of the ticker
	// and stop it when this method is called
}
