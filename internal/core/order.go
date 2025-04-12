package core

import (
	"fmt"
	"time"
)

// OrderFilter defines a function type for filtering orders
type OrderFilter func(order Order) bool

// SideType represents the direction of an order (BUY or SELL)
type SideType string

// OrderType represents the type of order (LIMIT, MARKET, etc.)
type OrderType string

// OrderStatusType represents the status of an order (NEW, FILLED, etc.)
type OrderStatusType string

// Order side constants
const (
	SideTypeBuy  SideType = "BUY"
	SideTypeSell SideType = "SELL"
)

// Order type constants
const (
	OrderTypeLimit           OrderType = "LIMIT"
	OrderTypeMarket          OrderType = "MARKET"
	OrderTypeLimitMaker      OrderType = "LIMIT_MAKER"
	OrderTypeStopLoss        OrderType = "STOP_LOSS"
	OrderTypeStopLossLimit   OrderType = "STOP_LOSS_LIMIT"
	OrderTypeTakeProfit      OrderType = "TAKE_PROFIT"
	OrderTypeTakeProfitLimit OrderType = "TAKE_PROFIT_LIMIT"
)

// Order status constants
const (
	OrderStatusTypeNew             OrderStatusType = "NEW"
	OrderStatusTypePartiallyFilled OrderStatusType = "PARTIALLY_FILLED"
	OrderStatusTypeFilled          OrderStatusType = "FILLED"
	OrderStatusTypeCanceled        OrderStatusType = "CANCELED"
	OrderStatusTypePendingCancel   OrderStatusType = "PENDING_CANCEL"
	OrderStatusTypeRejected        OrderStatusType = "REJECTED"
	OrderStatusTypeExpired         OrderStatusType = "EXPIRED"
)

// Order represents a trading order with its properties and status
type Order struct {
	ID         int64           `db:"id" json:"id" gorm:"primaryKey,autoIncrement"`
	ExchangeID int64           `db:"exchange_id" json:"exchange_id"`
	Pair       string          `db:"pair" json:"pair"`
	Side       SideType        `db:"side" json:"side"`
	Type       OrderType       `db:"type" json:"type"`
	Status     OrderStatusType `db:"status" json:"status"`
	Price      float64         `db:"price" json:"price"`
	Quantity   float64         `db:"quantity" json:"quantity"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`

	// OCO (One-Cancels-the-Other) Orders properties
	Stop    *float64 `db:"stop" json:"stop"`
	GroupID *int64   `db:"group_id" json:"group_id"`

	// Internal use for visualization and analysis
	RefPrice    float64 `json:"ref_price" gorm:"-"`
	Profit      float64 `json:"profit" gorm:"-"`
	ProfitValue float64 `json:"profit_value" gorm:"-"`
	Candle      Candle  `json:"-" gorm:"-"`
}

// GetID returns the order ID
func (o Order) GetID() int64 {
	return o.ID
}

// GetExchangeID returns the exchange ID
func (o Order) GetExchangeID() int64 {
	return o.ExchangeID
}

// GetPair returns the trading pair
func (o Order) GetPair() string {
	return o.Pair
}

// GetSide returns the order side (BUY or SELL)
func (o Order) GetSide() SideType {
	return o.Side
}

// GetType returns the order type
func (o Order) GetType() OrderType {
	return o.Type
}

// GetStatus returns the order status
func (o Order) GetStatus() OrderStatusType {
	return o.Status
}

// GetPrice returns the order price
func (o Order) GetPrice() float64 {
	return o.Price
}

// GetQuantity returns the order quantity
func (o Order) GetQuantity() float64 {
	return o.Quantity
}

// GetCreatedAt returns the order creation time
func (o Order) GetCreatedAt() time.Time {
	return o.CreatedAt
}

// GetUpdatedAt returns the order last update time
func (o Order) GetUpdatedAt() time.Time {
	return o.UpdatedAt
}

// GetStop returns the stop price for stop orders
func (o Order) GetStop() *float64 {
	return o.Stop
}

// GetGroupID returns the group ID for OCO orders
func (o Order) GetGroupID() *int64 {
	return o.GroupID
}

// GetRefPrice returns the reference price
func (o Order) GetRefPrice() float64 {
	return o.RefPrice
}

// GetProfit returns the profit percentage
func (o Order) GetProfit() float64 {
	return o.Profit
}

// GetProfitValue returns the profit value
func (o Order) GetProfitValue() float64 {
	return o.ProfitValue
}

// GetCandle returns the associated candle
func (o Order) GetCandle() Candle {
	return o.Candle
}

// GetValue returns the total value of the order (price * quantity)
func (o Order) GetValue() float64 {
	return o.Price * o.Quantity
}

// IsBuy returns true if the order is a buy order
func (o Order) IsBuy() bool {
	return o.Side == SideTypeBuy
}

// IsSell returns true if the order is a sell order
func (o Order) IsSell() bool {
	return o.Side == SideTypeSell
}

// IsActive returns true if the order is in an active state
func (o Order) IsActive() bool {
	return o.Status == OrderStatusTypeNew || o.Status == OrderStatusTypePartiallyFilled
}

// IsFilled returns true if the order is completely filled
func (o Order) IsFilled() bool {
	return o.Status == OrderStatusTypeFilled
}

// String returns a human-readable representation of the order
func (o Order) String() string {
	return fmt.Sprintf("[%s] %s %s | ID: %d, Type: %s, %f x $%f (~$%.2f)",
		o.Status, o.Side, o.Pair, o.ID, o.Type, o.Quantity, o.Price, o.Quantity*o.Price)
}
