package core

import (
	"context"
	"slices"
	"time"
)

// Storage defines the interface for order storage operations
type Storage interface {
	// CreateOrder stores a new order
	CreateOrder(ctx context.Context, order *Order) error

	// UpdateOrder updates an existing order
	UpdateOrder(ctx context.Context, order *Order) error

	// Orders retrieves orders based on provided filters
	Orders(ctx context.Context, filters ...OrderFilter) ([]*Order, error)
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
