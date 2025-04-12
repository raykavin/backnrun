package core

import (
	"slices"
	"time"
)

// OrderStorage defines the interface for order storage operations
type OrderStorage interface {
	// CreateOrder stores a new order
	CreateOrder(order *Order) error

	// UpdateOrder updates an existing order
	UpdateOrder(order *Order) error

	// Orders retrieves orders based on provided filters
	Orders(filters ...OrderFilter) ([]*Order, error)
}

func WithStatusIn(status ...OrderStatusType) OrderFilter {
	return func(order Order) bool {
		return slices.Contains(status, order.Status)
	}
}

func WithStatus(status OrderStatusType) OrderFilter {
	return func(order Order) bool {
		return order.Status == status
	}
}

func WithPair(pair string) OrderFilter {
	return func(order Order) bool {
		return order.Pair == pair
	}
}

func WithUpdateAtBeforeOrEqual(time time.Time) OrderFilter {
	return func(order Order) bool {
		return !order.UpdatedAt.After(time)
	}
}
