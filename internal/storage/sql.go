// File: storage/sql.go
package storage

import (
	"fmt"
	"time"

	"github.com/raykavin/backnrun/internal/core"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

// SQLStorage implements the core.OrderStorage interface using a SQL database via GORM
type SQLStorage struct {
	db *gorm.DB
}

// FromSQL creates a new SQL storage instance
func FromSQL(dialect gorm.Dialector, opts ...gorm.Option) (core.OrderStorage, error) {
	db, err := gorm.Open(dialect, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Configure connection pooling parameters
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Auto migrate the order model
	err = db.AutoMigrate(&core.Order{})
	if err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &SQLStorage{
		db: db,
	}, nil
}

// CreateOrder creates a new order in the SQL database
func (s *SQLStorage) CreateOrder(order *core.Order) error {
	result := s.db.Create(order)
	if result.Error != nil {
		return fmt.Errorf("failed to create order: %w", result.Error)
	}

	return nil
}

// UpdateOrder updates an existing order in the SQL database
func (s *SQLStorage) UpdateOrder(order *core.Order) error {
	// First check if the order exists
	var existing core.Order
	result := s.db.First(&existing, order.ID)
	if result.Error != nil {
		return fmt.Errorf("order not found: %w", result.Error)
	}

	// Update the order
	result = s.db.Save(order)
	if result.Error != nil {
		return fmt.Errorf("failed to update order: %w", result.Error)
	}

	return nil
}

// Orders retrieves orders from the SQL database based on provided filters
func (s *SQLStorage) Orders(filters ...core.OrderFilter) ([]*core.Order, error) {
	var orders []*core.Order

	// Fetch all orders from the database
	result := s.db.Find(&orders)
	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to fetch orders: %w", result.Error)
	}

	// Apply filters in memory
	// Note: This could be optimized by translating filters to SQL WHERE clauses
	filteredOrders := lo.Filter(orders, func(order *core.Order, _ int) bool {
		for _, filter := range filters {
			if !filter(*order) {
				return false
			}
		}
		return true
	})

	return filteredOrders, nil
}

// Close closes the database connection
func (s *SQLStorage) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	return sqlDB.Close()
}

// WithTransaction executes the given function within a database transaction
func (s *SQLStorage) WithTransaction(fn func(tx *gorm.DB) error) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}

// OrdersWithQuery allows for more customized querying using GORM's query builder
func (s *SQLStorage) OrdersWithQuery(query func(*gorm.DB) *gorm.DB) ([]*core.Order, error) {
	var orders []*core.Order

	result := query(s.db).Find(&orders)
	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to execute query: %w", result.Error)
	}

	return orders, nil
}
