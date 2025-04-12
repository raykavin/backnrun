package plot

import (
	"fmt"
	"strings"

	"github.com/StudioSol/set"
	"github.com/raykavin/backnrun/internal/core"
)

// OnOrder handles new order events
func (c *Chart) OnOrder(order core.Order) {
	c.Lock()
	defer c.Unlock()

	// Initialize order set for pair if needed
	if c.ordersIDsByPair[order.Pair] == nil {
		c.ordersIDsByPair[order.Pair] = set.NewLinkedHashSetINT64()
	}

	c.ordersIDsByPair[order.Pair].Add(order.ID)
	c.orderByID[order.ID] = order
}

// shapesByPair returns shapes for stop-loss and limit orders for a trading pair
func (c *Chart) shapesByPair(pair string) []Shape {
	shapes := make([]Shape, 0)

	if _, ok := c.ordersIDsByPair[pair]; !ok {
		return shapes
	}

	for id := range c.ordersIDsByPair[pair].Iter() {
		order := c.orderByID[id]

		// Only generate shapes for stop-loss and limit-maker orders
		if order.Type != core.OrderTypeStopLoss &&
			order.Type != core.OrderTypeLimitMaker {
			continue
		}

		shape := Shape{
			StartX: order.CreatedAt,
			EndX:   order.UpdatedAt,
			StartY: order.RefPrice,
			EndY:   order.Price,
			Color:  "rgba(0, 255, 0, 0.3)", // Default to green for limit orders
		}

		// Use red for stop-loss orders
		if order.Type == core.OrderTypeStopLoss {
			shape.Color = "rgba(255, 0, 0, 0.3)"
		}

		shapes = append(shapes, shape)
	}

	return shapes
}

// orderStringByPair returns order data as string arrays for a trading pair
func (c *Chart) orderStringByPair(pair string) [][]string {
	orders := make([][]string, 0)

	if _, ok := c.ordersIDsByPair[pair]; !ok {
		return orders
	}

	for id := range c.ordersIDsByPair[pair].Iter() {
		o := c.orderByID[id]

		// Format profit if present
		var profit string
		if o.Profit != 0 {
			profit = fmt.Sprintf("%.2f", o.Profit)
		}

		// Create CSV row
		orderString := fmt.Sprintf("%s,%s,%s,%d,%s,%f,%f,%.2f,%s",
			o.CreatedAt, o.Status, o.Side, o.ID, o.Type, o.Quantity, o.Price, o.Quantity*o.Price, profit)
		order := strings.Split(orderString, ",")
		orders = append(orders, order)
	}

	return orders
}
