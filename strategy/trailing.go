// Package strategy provides trading strategy implementations.
package strategy

// TrailingStop implements a trailing stop mechanism for trading systems.
// It tracks a price and automatically adjusts the stop level as the price moves favorably.
type TrailingStop struct {
	currentPrice float64 // The most recent price
	stopLevel    float64 // The stop level price
	isActive     bool    // Whether the trailing stop is currently active
}

// NewTrailingStop creates a new TrailingStop instance.
func NewTrailingStop() *TrailingStop {
	return &TrailingStop{}
}

// Start activates the trailing stop with the given current price and initial stop level.
func (t *TrailingStop) Start(currentPrice, stopLevel float64) {
	t.currentPrice = currentPrice
	t.stopLevel = stopLevel
	t.isActive = true
}

// Stop deactivates the trailing stop.
func (t *TrailingStop) Stop() {
	t.isActive = false
}

// Active returns whether the trailing stop is currently active.
func (t *TrailingStop) Active() bool {
	return t.isActive
}

// Update updates the trailing stop with a new price and returns true if the stop was triggered.
// The stop is triggered when the current price falls below or equals the stop level.
// If the price increases above the previous price, the stop level is adjusted upward by the same amount.
func (t *TrailingStop) Update(newPrice float64) bool {
	if !t.isActive {
		return false
	}

	// If price moved higher, adjust the stop level proportionally
	if newPrice > t.currentPrice {
		priceIncrease := newPrice - t.currentPrice
		t.stopLevel += priceIncrease
	}

	// Update the current price
	t.currentPrice = newPrice

	// Return true if stop is triggered (price at or below stop level)
	return newPrice <= t.stopLevel
}
