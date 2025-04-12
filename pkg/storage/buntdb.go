package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync/atomic"

	"github.com/raykavin/backnrun/pkg/core"
	"github.com/tidwall/buntdb"
)

// BuntStorage implements the core.OrderStorage interface using BuntDB
type BuntStorage struct {
	lastID int64
	db     *buntdb.DB
}

// FromMemory creates an in-memory storage
func FromMemory() (core.OrderStorage, error) {
	return NewBuntStorage(":memory:")
}

// FromFile creates a file-based storage
func FromFile(file string) (core.OrderStorage, error) {
	return NewBuntStorage(file)
}

// NewBuntStorage creates a new BuntDB storage instance
func NewBuntStorage(sourceFile string) (core.OrderStorage, error) {
	db, err := buntdb.Open(sourceFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open buntdb: %w", err)
	}

	err = db.CreateIndex("update_index", "*", buntdb.IndexJSON("updated_at"))
	if err != nil {
		return nil, fmt.Errorf("failed to create index: %w", err)
	}

	return &BuntStorage{
		db: db,
	}, nil
}

// getID generates a unique ID for orders
func (b *BuntStorage) getID() int64 {
	return atomic.AddInt64(&b.lastID, 1)
}

// CreateOrder stores a new order in the database
func (b *BuntStorage) CreateOrder(order *core.Order) error {
	return b.db.Update(func(tx *buntdb.Tx) error {
		order.ID = b.getID()
		content, err := json.Marshal(order)
		if err != nil {
			return fmt.Errorf("failed to marshal order: %w", err)
		}

		_, _, err = tx.Set(strconv.FormatInt(order.ID, 10), string(content), nil)
		if err != nil {
			return fmt.Errorf("failed to store order: %w", err)
		}

		return nil
	})
}

// UpdateOrder updates an existing order in the database
func (b *BuntStorage) UpdateOrder(order *core.Order) error {
	return b.db.Update(func(tx *buntdb.Tx) error {
		id := strconv.FormatInt(order.ID, 10)

		// Check if order exists
		_, err := tx.Get(id)
		if err != nil {
			return fmt.Errorf("order not found: %w", err)
		}

		content, err := json.Marshal(order)
		if err != nil {
			return fmt.Errorf("failed to marshal order: %w", err)
		}

		_, _, err = tx.Set(id, string(content), nil)
		if err != nil {
			return fmt.Errorf("failed to update order: %w", err)
		}

		return nil
	})
}

// Orders retrieves orders from the database based on provided filters
func (b *BuntStorage) Orders(filters ...core.OrderFilter) ([]*core.Order, error) {
	orders := make([]*core.Order, 0)

	err := b.db.View(func(tx *buntdb.Tx) error {
		err := tx.Ascend("update_index", func(_, value string) bool {
			var order core.Order
			err := json.Unmarshal([]byte(value), &order)
			if err != nil {
				log.Printf("Failed to unmarshal order: %v", err)
				return true // Continue iteration
			}

			// Apply all filters
			for _, filter := range filters {
				if !filter(order) {
					return true // Skip this order and continue iteration
				}
			}

			// All filters passed, add this order
			orders = append(orders, &order)
			return true
		})

		if err != nil {
			return fmt.Errorf("failed to iterate over orders: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return orders, nil
}

// Close closes the database connection
func (b *BuntStorage) Close() error {
	if b.db != nil {
		return b.db.Close()
	}
	return nil
}
