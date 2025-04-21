package order

import (
	"math"
	"time"

	"github.com/raykavin/backnrun/core"
)

// TradeResult contains the outcome of a completed trade
type TradeResult struct {
	Pair          string
	ProfitPercent float64
	ProfitValue   float64
	Side          core.SideType
	Duration      time.Duration
	CreatedAt     time.Time
}

// Position represents a current trading position
type Position struct {
	Side      core.SideType
	CreatedAt time.Time
	AvgPrice  float64
	Quantity  float64
}

// Update modifies the position based on a new order
// Returns a trade result if the order closes or partially closes the position
func (p *Position) Update(order *core.Order) (result *TradeResult, finished bool) {
	// Get the effective price considering stop orders
	price := order.Price
	if order.Type == core.OrderTypeStopLoss || order.Type == core.OrderTypeStopLossLimit {
		price = *order.Stop
	}

	// If the order is on the same side as the position, increase position size
	if p.Side == order.Side {
		// Calculate new average price
		p.AvgPrice = calculateWeightedAverage(p.AvgPrice, p.Quantity, price, order.Quantity)
		p.Quantity += order.Quantity
		return nil, false
	}

	// Order is closing or reducing the position
	var tradeResult *TradeResult
	var isPositionClosed bool

	if p.Quantity == order.Quantity {
		// Position fully closed
		isPositionClosed = true
	} else if p.Quantity > order.Quantity {
		// Position partially closed
		p.Quantity -= order.Quantity
	} else {
		// Position reversed
		remainingQuantity := order.Quantity - p.Quantity
		p.Quantity = remainingQuantity
		p.Side = order.Side
		p.CreatedAt = order.CreatedAt
		p.AvgPrice = price
	}

	// Calculate profit for the closed portion
	closedQuantity := math.Min(p.Quantity, order.Quantity)
	profitPercent := (price - p.AvgPrice) / p.AvgPrice
	profitValue := (price - p.AvgPrice) * closedQuantity

	// Update order with profit information
	order.Profit = profitPercent
	order.ProfitValue = profitValue

	// Create trade result
	tradeResult = &TradeResult{
		CreatedAt:     order.CreatedAt,
		Pair:          order.Pair,
		Duration:      order.CreatedAt.Sub(p.CreatedAt),
		ProfitPercent: profitPercent,
		ProfitValue:   profitValue,
		Side:          p.Side,
	}

	return tradeResult, isPositionClosed
}

// calculateWeightedAverage computes the weighted average of two price-quantity pairs
func calculateWeightedAverage(price1, quantity1, price2, quantity2 float64) float64 {
	return (price1*quantity1 + price2*quantity2) / (quantity1 + quantity2)
}
