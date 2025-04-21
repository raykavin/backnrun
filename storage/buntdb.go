package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync/atomic"

	"github.com/raykavin/backnrun/core"
	"github.com/tidwall/buntdb"
)

const (
	// DefaultIndexName is the default index used for order retrieval
	DefaultIndexName = "update_index"
)

// BuntStorage implements the core.Storage interface using BuntDB
type BuntStorage struct {
	lastID int64
	db     *buntdb.DB
}

// BuntConfig holds configuration options for BuntDB
type BuntConfig struct {
	// Additional indexes to create beyond the default update_index
	AdditionalIndexes map[string]string
	// SyncPolicy determines how often data is synchronized to disk
	SyncPolicy buntdb.SyncPolicy
}

// DefaultBuntConfig returns the default configuration for BuntDB
func DefaultBuntConfig() BuntConfig {
	return BuntConfig{
		AdditionalIndexes: make(map[string]string),
		SyncPolicy:        buntdb.Never,
	}
}

// NewFromMemory creates an in-memory storage with default configuration
func NewFromMemory() (core.Storage, error) {
	return NewBuntStorage(":memory:", DefaultBuntConfig())
}

// NewFromFile creates a file-based storage with default configuration
func NewFromFile(file string) (core.Storage, error) {
	return NewBuntStorage(file, DefaultBuntConfig())
}

// NewBuntStorage creates a new BuntDB storage instance with the specified configuration
func NewBuntStorage(sourceFile string, config BuntConfig) (core.Storage, error) {
	db, err := buntdb.Open(sourceFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open buntdb: %w", err)
	}

	// Apply configuration
	if err := db.SetConfig(buntdb.Config{
		SyncPolicy: config.SyncPolicy,
	}); err != nil {
		return nil, fmt.Errorf("failed to configure buntdb: %w", err)
	}

	// Create default index for ordering by update timestamp
	if err := db.CreateIndex(DefaultIndexName, "*", buntdb.IndexJSON("updated_at")); err != nil {
		return nil, fmt.Errorf("failed to create default index: %w", err)
	}

	// Create any additional indexes from the configuration
	for name, pattern := range config.AdditionalIndexes {
		if err := db.CreateIndex(name, "*", buntdb.IndexJSON(pattern)); err != nil {
			return nil, fmt.Errorf("failed to create index %s: %w", name, err)
		}
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
func (b *BuntStorage) CreateOrder(_ context.Context, order *core.Order) error {
	// Use a context-aware version if BuntDB adds context support in future
	return b.db.Update(func(tx *buntdb.Tx) error {
		if order.ID == 0 {
			order.ID = b.getID()
		}

		content, err := json.Marshal(order)
		if err != nil {
			return fmt.Errorf("failed to marshal order: %w", err)
		}

		key := strconv.FormatInt(order.ID, 10)
		_, _, err = tx.Set(key, string(content), nil)
		if err != nil {
			return fmt.Errorf("failed to store order: %w", err)
		}

		return nil
	})
}

// UpdateOrder updates an existing order in the database
func (b *BuntStorage) UpdateOrder(_ context.Context, order *core.Order) error {
	// Use a context-aware version if BuntDB adds context support in future
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
func (b *BuntStorage) Orders(_ context.Context, filters ...core.OrderFilter) ([]*core.Order, error) {
	orders := make([]*core.Order, 0)

	// Use a context-aware version if BuntDB adds context support in future
	err := b.db.View(func(tx *buntdb.Tx) error {
		err := tx.Ascend(DefaultIndexName, func(key, value string) bool {
			var order core.Order
			if err := json.Unmarshal([]byte(value), &order); err != nil {
				log.Printf("Failed to unmarshal order %s: %v", key, err)
				return true // Continue iteration
			}

			// Skip filtering if no filters provided
			if len(filters) == 0 {
				orders = append(orders, &order)
				return true
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
		return nil, fmt.Errorf("failed to query orders: %w", err)
	}

	return orders, nil
}

// OrdersWithIndex retrieves orders using a specific index
func (b *BuntStorage) OrdersWithIndex(_ context.Context, indexName string, filters ...core.OrderFilter) ([]*core.Order, error) {
	orders := make([]*core.Order, 0)

	err := b.db.View(func(tx *buntdb.Tx) error {
		err := tx.Ascend(indexName, func(key, value string) bool {
			var order core.Order
			if err := json.Unmarshal([]byte(value), &order); err != nil {
				log.Printf("Failed to unmarshal order %s: %v", key, err)
				return true
			}

			// Apply all filters
			for _, filter := range filters {
				if !filter(order) {
					return true
				}
			}

			// All filters passed, add this order
			orders = append(orders, &order)
			return true
		})

		if err != nil {
			return fmt.Errorf("failed to iterate over orders with index %s: %w", indexName, err)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to query orders with index: %w", err)
	}

	return orders, nil
}

// WithTransaction executes operations in a transaction
// Note: This is a simplified version as BuntDB's transaction model is different from SQL databases
func (b *BuntStorage) WithTransaction(ctx context.Context, fn func(tx any) error) error {
	return b.db.Update(func(tx *buntdb.Tx) error {
		return fn(tx)
	})
}

// Close closes the database connection
func (b *BuntStorage) Close() error {
	if b.db != nil {
		return b.db.Close()
	}
	return nil
}

// Legacy function aliases for backward compatibility

// FromMemory creates an in-memory storage (legacy function)
func FromMemory() (core.Storage, error) {
	return NewFromMemory()
}

// FromFile creates a file-based storage (legacy function)
func FromFile(file string) (core.Storage, error) {
	return NewFromFile(file)
}
