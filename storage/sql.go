// File: storage/sql.go
package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/raykavin/backnrun/core"
	"github.com/samber/lo"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SQLStorage implements the core.Storage interface using a SQL database via GORM
type SQLStorage struct {
	db *gorm.DB
}

// Config holds the configuration for SQL database connections
type Config struct {
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
}

// DefaultConfig returns a default configuration for SQL connections
func DefaultConfig() Config {
	return Config{
		MaxIdleConns:    5,
		MaxOpenConns:    10,
		ConnMaxLifetime: time.Hour,
	}
}

// NewFromSQLite creates a new SQLite storage instance
func NewFromSQLite(dbPath string, config Config, opts ...gorm.Option) (core.Storage, error) {
	dialect := sqlite.Open(dbPath)
	return newFromSQL(dialect, config, opts...)
}

// newFromSQL creates a new SQL storage instance with the specified configuration
func newFromSQL(dialect gorm.Dialector, config Config, opts ...gorm.Option) (core.Storage, error) {
	db, err := gorm.Open(dialect, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Apply configuration
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)

	// Auto migrate the order model
	if err = db.AutoMigrate(&core.Order{}); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &SQLStorage{db: db}, nil
}

// CreateOrder creates a new order in the SQL database
func (s *SQLStorage) CreateOrder(ctx context.Context, order *core.Order) error {
	tx := s.db.WithContext(ctx)
	if result := tx.Create(order); result.Error != nil {
		return fmt.Errorf("failed to create order: %w", result.Error)
	}
	return nil
}

// UpdateOrder updates an existing order in the SQL database
func (s *SQLStorage) UpdateOrder(ctx context.Context, order *core.Order) error {
	tx := s.db.WithContext(ctx)

	// Check if the order exists
	var existing core.Order
	if result := tx.First(&existing, order.ID); result.Error != nil {
		return fmt.Errorf("order not found: %w", result.Error)
	}

	// Update the order
	if result := tx.Save(order); result.Error != nil {
		return fmt.Errorf("failed to update order: %w", result.Error)
	}

	return nil
}

// Orders retrieves orders from the SQL database based on provided filters
func (s *SQLStorage) Orders(ctx context.Context, filters ...core.OrderFilter) ([]*core.Order, error) {
	tx := s.db.WithContext(ctx)

	// Start building the query
	query := tx

	// Apply filters directly to the query if possible
	// This is a placeholder for future optimization
	// For now, we'll still apply filters in memory

	var orders []*core.Order
	if result := query.Find(&orders); result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to fetch orders: %w", result.Error)
	}

	// Apply filters in memory
	if len(filters) > 0 {
		orders = lo.Filter(orders, func(order *core.Order, _ int) bool {
			for _, filter := range filters {
				if !filter(*order) {
					return false
				}
			}
			return true
		})
	}

	return orders, nil
}

// OrdersWithQuery allows for more customized querying using GORM's query builder
func (s *SQLStorage) OrdersWithQuery(ctx context.Context, queryFn func(*gorm.DB) *gorm.DB) ([]*core.Order, error) {
	tx := s.db.WithContext(ctx)

	var orders []*core.Order
	result := queryFn(tx).Find(&orders)

	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to execute query: %w", result.Error)
	}

	return orders, nil
}

// WithTransaction executes the given function within a database transaction
func (s *SQLStorage) WithTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return s.db.WithContext(ctx).Transaction(fn)
}

// Close closes the database connection
func (s *SQLStorage) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}
	return sqlDB.Close()
}

// Legacy function aliases for backward compatibility
// These can be removed once all code is updated to use the new functions

// FromSQLite creates a new SQLite storage instance (legacy function)
func FromSQLite(dbPath string, opts ...gorm.Option) (core.Storage, error) {
	return NewFromSQLite(dbPath, DefaultConfig(), opts...)
}

// FromSQL creates a new SQL storage instance (legacy function)
func FromSQL(dialect gorm.Dialector, opts ...gorm.Option) (core.Storage, error) {
	return newFromSQL(dialect, DefaultConfig(), opts...)
}
