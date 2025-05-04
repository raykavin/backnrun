// Package exchange provides functionality for interacting with cryptocurrency exchanges
// and simulating trading activities.
package exchange

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/raykavin/backnrun/core"

	"github.com/adshao/go-binance/v2/common"
)

// ---------------------
// Types
// ---------------------

// AssetValue represents the value of an asset at a specific time
type AssetValue struct {
	Time  time.Time
	Value float64
}

// assetInfo represents balance information of an asset
type assetInfo struct {
	Free float64
	Lock float64
}

// PaperWallet implements a simulated wallet for backtesting
type PaperWallet struct {
	mu sync.RWMutex

	// Context and configuration
	ctx          context.Context
	baseCoin     string
	takerFee     float64
	makerFee     float64
	initialValue float64
	counter      atomic.Int64
	feeder       core.Feeder

	// Wallet data
	orders        []core.Order
	assets        map[string]*assetInfo
	avgShortPrice map[string]float64
	avgLongPrice  map[string]float64
	volume        map[string]float64

	// Candle data
	lastCandle map[string]core.Candle
	fistCandle map[string]core.Candle

	// Value history
	assetValues  map[string][]AssetValue
	equityValues []AssetValue

	log core.Logger
}

// PaperWalletOption defines an option function to configure PaperWallet
type PaperWalletOption func(*PaperWallet)

// ---------------------
// Configuration Options
// ---------------------

// WithPaperAsset adds an initial asset to the wallet
func WithPaperAsset(pair string, amount float64) PaperWalletOption {
	return func(wallet *PaperWallet) {
		wallet.assets[pair] = &assetInfo{
			Free: amount,
			Lock: 0,
		}
	}
}

// WithPaperFee configures the wallet fees
func WithPaperFee(maker, taker float64) PaperWalletOption {
	return func(wallet *PaperWallet) {
		wallet.makerFee = maker
		wallet.takerFee = taker
	}
}

// WithDataFeed configures the data provider
func WithDataFeed(feeder core.Feeder) PaperWalletOption {
	return func(wallet *PaperWallet) {
		wallet.feeder = feeder
	}
}

// ---------------------
// Constructor
// ---------------------

// NewPaperWallet creates a new simulated wallet
func NewPaperWallet(ctx context.Context, baseCoin string, log core.Logger, options ...PaperWalletOption) *PaperWallet {
	wallet := PaperWallet{
		ctx:           ctx,
		baseCoin:      baseCoin,
		log:           log,
		orders:        make([]core.Order, 0),
		assets:        make(map[string]*assetInfo),
		fistCandle:    make(map[string]core.Candle),
		lastCandle:    make(map[string]core.Candle),
		avgShortPrice: make(map[string]float64),
		avgLongPrice:  make(map[string]float64),
		volume:        make(map[string]float64),
		assetValues:   make(map[string][]AssetValue),
		equityValues:  make([]AssetValue, 0),
	}

	// Apply options
	for _, option := range options {
		option(&wallet)
	}

	// Initialize initial wallet value
	wallet.initialValue = wallet.getAssetFreeAmount(wallet.baseCoin)

	log.Info("Using paper wallet")
	log.Infof("Initial Portfolio = %f %s", wallet.initialValue, wallet.baseCoin)

	return &wallet
}

// ---------------------
// Basic Methods
// ---------------------

// ID generates a unique ID for orders
func (p *PaperWallet) ID() int64 {
	return p.counter.Add(1)
}

// AssetsInfo returns information about the assets of a pair
func (p *PaperWallet) AssetsInfo(pair string) (core.AssetInfo, error) {
	asset, quote := SplitAssetQuote(pair)
	return core.NewAssetInfo(
		asset,
		quote,
		0,
		math.MaxFloat64,
		0,
		math.MaxFloat64,
		0.00000001,
		0.00000001,
		8,
		8,
	)
}

// Pairs returns the list of available pairs in the wallet
func (p *PaperWallet) Pairs() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	pairs := make([]string, 0, len(p.assets))
	for pair := range p.assets {
		pairs = append(pairs, pair)
	}
	return pairs
}

// LastQuote returns the last quote of a pair
func (p *PaperWallet) LastQuote(ctx context.Context, pair string) (float64, error) {
	return p.feeder.LastQuote(ctx, pair)
}

// ---------------------
// Asset Management
// ---------------------

// AssetValues returns the value history of an asset
func (p *PaperWallet) AssetValues(pair string) []AssetValue {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.assetValues[pair]
}

// EquityValues returns the wallet's value history
func (p *PaperWallet) EquityValues() []AssetValue {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.equityValues
}

// getAssetFreeAmount returns the free balance of an asset
func (p *PaperWallet) getAssetFreeAmount(asset string) float64 {
	assetInfo, ok := p.assets[asset]
	if !ok {
		return 0
	}
	return assetInfo.Free
}

// getAssetTotalAmount returns the total balance (free + locked) of an asset
func (p *PaperWallet) getAssetTotalAmount(asset string) float64 {
	assetInfo, ok := p.assets[asset]
	if !ok {
		return 0
	}
	return assetInfo.Free + assetInfo.Lock
}

// ensureAssetExists ensures that an asset exists in the wallet
func (p *PaperWallet) ensureAssetExists(asset string) {
	if _, ok := p.assets[asset]; !ok {
		p.assets[asset] = &assetInfo{}
	}
}

// ---------------------
// Performance Analysis
// ---------------------

// MaxDrawdown calculates the maximum drawdown of the wallet
func (p *PaperWallet) MaxDrawdown() (float64, time.Time, time.Time) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.equityValues) < 1 {
		return 0, time.Time{}, time.Time{}
	}

	localMin := math.MaxFloat64
	localMinBase := p.equityValues[0].Value
	localMinStart := p.equityValues[0].Time
	localMinEnd := p.equityValues[0].Time

	globalMin := localMin
	globalMinBase := localMinBase
	globalMinStart := localMinStart
	globalMinEnd := localMinEnd

	for i := 1; i < len(p.equityValues); i++ {
		diff := p.equityValues[i].Value - p.equityValues[i-1].Value

		if localMin > 0 {
			localMin = diff
			localMinBase = p.equityValues[i-1].Value
			localMinStart = p.equityValues[i-1].Time
			localMinEnd = p.equityValues[i].Time
		} else {
			localMin += diff
			localMinEnd = p.equityValues[i].Time
		}

		if localMin < globalMin {
			globalMin = localMin
			globalMinBase = localMinBase
			globalMinStart = localMinStart
			globalMinEnd = localMinEnd
		}
	}

	return globalMin / globalMinBase, globalMinStart, globalMinEnd
}

// Summary prints a summary of the wallet
func (p *PaperWallet) Summary() {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var (
		total        float64
		marketChange float64
		volume       float64
	)

	fmt.Println("----- FINAL WALLET -----")

	// Calculate total asset value
	for pair := range p.lastCandle {
		asset, quote := SplitAssetQuote(pair)
		assetInfo, ok := p.assets[asset]
		if !ok {
			continue
		}

		quantity := assetInfo.Free + assetInfo.Lock

		// Calculate asset value
		value := p.calculateAssetValue(pair, asset, quantity)
		total += value

		// Calculate market change
		marketChange += p.calculateMarketChange(pair)

		fmt.Printf("%.4f %s = %.4f %s\n", quantity, asset, value, quote)
	}

	// Calculate average market change
	avgMarketChange := marketChange / float64(len(p.lastCandle))

	// Calculate base currency balance
	baseCoinValue := p.getAssetTotalAmount(p.baseCoin)

	// Calculate profit
	profit := total + baseCoinValue - p.initialValue

	// Print base currency information
	fmt.Printf("%.4f %s\n", baseCoinValue, p.baseCoin)
	fmt.Println()

	// Calculate maximum drawdown
	maxDrawDown, _, _ := p.MaxDrawdown()

	// Print returns summary
	fmt.Println("----- RETURNS -----")
	fmt.Printf("START PORTFOLIO     = %.2f %s\n", p.initialValue, p.baseCoin)
	fmt.Printf("FINAL PORTFOLIO     = %.2f %s\n", total+baseCoinValue, p.baseCoin)
	fmt.Printf("GROSS PROFIT        =  %f %s (%.2f%%)\n", profit, p.baseCoin, profit/p.initialValue*100)
	fmt.Printf("MARKET CHANGE (B&H) =  %.2f%%\n", avgMarketChange*100)
	fmt.Println()

	// Print risk information
	fmt.Println("------ RISK -------")
	fmt.Printf("MAX DRAWDOWN = %.2f %%\n", maxDrawDown*100)
	fmt.Println()

	// Print volume information
	fmt.Println("------ VOLUME -----")
	for pair, vol := range p.volume {
		volume += vol
		fmt.Printf("%s         = %.2f %s\n", pair, vol, p.baseCoin)
	}
	fmt.Printf("TOTAL           = %.2f %s\n", volume, p.baseCoin)
	fmt.Println("-------------------")
}

// calculateAssetValue calculates the value of an asset
func (p *PaperWallet) calculateAssetValue(pair, asset string, quantity float64) float64 {
	if quantity == 0 {
		return 0
	}

	// If the quantity is positive, it's a long position
	if quantity > 0 {
		return quantity * p.lastCandle[pair].Close
	}

	// If the quantity is negative, it's a short position
	// Calculate the total value of the short position
	totalShort := 2.0*p.avgShortPrice[pair]*quantity - p.lastCandle[pair].Close*quantity
	return math.Abs(totalShort)
}

// calculateMarketChange calculates the price change of a pair
func (p *PaperWallet) calculateMarketChange(pair string) float64 {
	firstPrice := p.fistCandle[pair].Close
	lastPrice := p.lastCandle[pair].Close
	return (lastPrice - firstPrice) / firstPrice
}

// ---------------------
// Fund Validation
// ---------------------

// validateFunds verifies if there are sufficient funds for an operation
// Note: This function assumes the mutex is already locked by the caller
func (p *PaperWallet) validateFunds(side core.SideType, pair string, amount, value float64, fill bool) error {
	asset, quote := SplitAssetQuote(pair)

	// Ensure assets exist
	p.ensureAssetExists(asset)
	p.ensureAssetExists(quote)

	// Check if there are sufficient funds for the operation
	if side == core.SideTypeSell {
		return p.validateSellFunds(pair, asset, quote, amount, value, fill)
	} else { // SideTypeBuy
		return p.validateBuyFunds(pair, asset, quote, amount, value, fill)
	}
}

// validateSellFunds verifies and processes funds for selling
// Note: This function assumes the mutex is already locked by the caller
func (p *PaperWallet) validateSellFunds(pair, asset, quote string, amount, value float64, fill bool) error {
	// Calculate available funds
	funds := p.assets[quote].Free
	if p.assets[asset].Free > 0 {
		funds += p.assets[asset].Free * value
	}

	// Check if there are sufficient funds
	if funds < (amount * value) {
		return &OrderError{
			Err:      ErrInsufficientFunds,
			Pair:     pair,
			Quantity: amount,
		}
	}

	// Calculate values to be locked
	lockedAsset := math.Min(math.Max(p.assets[asset].Free, 0), amount) // ignore negative values
	lockedQuote := (amount - lockedAsset) * value

	// Update balances
	p.assets[asset].Free -= lockedAsset
	p.assets[quote].Free -= lockedQuote

	if fill {
		// Update average price
		p.updateAveragePrice(core.SideTypeSell, pair, amount, value)

		if lockedQuote > 0 { // entering short position
			p.assets[asset].Free -= amount
		} else { // liquidating long position
			p.assets[quote].Free += amount * value
		}
	} else {
		// Lock values
		p.assets[asset].Lock += lockedAsset
		p.assets[quote].Lock += lockedQuote
	}

	p.log.Debugf("%s -> LOCK = %f / FREE %f", asset, p.assets[asset].Lock, p.assets[asset].Free)
	return nil
}

// validateBuyFunds verifies and processes funds for buying
// Note: This function assumes the mutex is already locked by the caller
func (p *PaperWallet) validateBuyFunds(pair, asset, quote string, amount, value float64, fill bool) error {
	var liquidShortValue float64

	// If there's a short position, calculate liquidation value
	if p.assets[asset].Free < 0 {
		v := math.Abs(p.assets[asset].Free)
		liquidShortValue = 2*v*p.avgShortPrice[pair] - v*value
		funds := p.assets[quote].Free + liquidShortValue

		// Calculate effective amount to buy
		amountToBuy := amount
		if p.assets[asset].Free < 0 {
			amountToBuy = amount + p.assets[asset].Free
		}

		// Check if there are sufficient funds
		if funds < (amountToBuy * value) {
			return &OrderError{
				Err:      ErrInsufficientFunds,
				Pair:     pair,
				Quantity: amount,
			}
		}

		// Calculate values to be locked
		lockedAsset := math.Min(-math.Min(p.assets[asset].Free, 0), amount)
		lockedQuote := (amount-lockedAsset)*value - liquidShortValue

		// Update balances
		p.assets[asset].Free += lockedAsset
		p.assets[quote].Free -= lockedQuote

		if fill {
			// Update average price
			p.updateAveragePrice(core.SideTypeBuy, pair, amount, value)
			p.assets[asset].Free += amount - lockedAsset
		} else {
			// Lock values
			p.assets[asset].Lock += lockedAsset
			p.assets[quote].Lock += lockedQuote
		}

		p.log.Debugf("%s -> LOCK = %f / FREE %f", asset, p.assets[asset].Lock, p.assets[asset].Free)
	} else {
		// Simple case: buy with quote balance
		if p.assets[quote].Free < amount*value {
			return &OrderError{
				Err:      ErrInsufficientFunds,
				Pair:     pair,
				Quantity: amount,
			}
		}

		if fill {
			// Update average price and balances directly
			p.updateAveragePrice(core.SideTypeBuy, pair, amount, value)
			p.assets[quote].Free -= amount * value
			p.assets[asset].Free += amount
		} else {
			// Lock values
			p.assets[quote].Lock += amount * value
			p.assets[quote].Free -= amount * value
		}
	}

	return nil
}

// updateAveragePrice updates the average buy/sell price
// Note: This function assumes the mutex is already locked by the caller
func (p *PaperWallet) updateAveragePrice(side core.SideType, pair string, amount, value float64) {
	actualQty := 0.0
	asset, quote := SplitAssetQuote(pair)

	if p.assets[asset] != nil {
		actualQty = p.assets[asset].Free
	}

	// No previous position
	if actualQty == 0 {
		if side == core.SideTypeBuy {
			p.avgLongPrice[pair] = value
		} else {
			p.avgShortPrice[pair] = value
		}
		return
	}

	// Long position + buy order
	if actualQty > 0 && side == core.SideTypeBuy {
		positionValue := p.avgLongPrice[pair] * actualQty
		p.avgLongPrice[pair] = (positionValue + amount*value) / (actualQty + amount)
		return
	}

	// Long position + sell order
	if actualQty > 0 && side == core.SideTypeSell {
		// Calculate profit
		profitValue := amount*value - math.Min(amount, actualQty)*p.avgLongPrice[pair]
		percentage := profitValue / (amount * p.avgLongPrice[pair])
		p.log.Infof("PROFIT = %.4f %s (%.2f %%)", profitValue, quote, percentage*100.0)

		// If the sold quantity doesn't close the position
		if amount <= actualQty {
			return
		}

		// If the sale exceeds the position, starts a short position
		p.avgShortPrice[pair] = value
		return
	}

	// Short position + sell order
	if actualQty < 0 && side == core.SideTypeSell {
		positionValue := p.avgShortPrice[pair] * -actualQty
		p.avgShortPrice[pair] = (positionValue + amount*value) / (-actualQty + amount)
		return
	}

	// Short position + buy order
	if actualQty < 0 && side == core.SideTypeBuy {
		// Calculate profit
		profitValue := math.Min(amount, -actualQty)*p.avgShortPrice[pair] - amount*value
		percentage := profitValue / (amount * p.avgShortPrice[pair])
		p.log.Infof("PROFIT = %.4f %s (%.2f %%)", profitValue, quote, percentage*100.0)

		// If the bought quantity doesn't close the position
		if amount <= -actualQty {
			return
		}

		// If the purchase exceeds the short position, starts a long position
		p.avgLongPrice[pair] = value
	}
}

// ---------------------
// Candle Processing
// ---------------------

// OnCandle processes a new candle
func (p *PaperWallet) OnCandle(candle core.Candle) {
	p.mu.Lock()

	// Update the most recent candle
	p.lastCandle[candle.Pair] = candle

	// Register the first candle, if it doesn't exist yet
	if _, ok := p.fistCandle[candle.Pair]; !ok {
		p.fistCandle[candle.Pair] = candle
	}

	// Create a local copy of orders to process to avoid holding the lock during processing
	ordersToProcess := make([]core.Order, len(p.orders))
	copy(ordersToProcess, p.orders)

	// Mutex can be unlocked now for order processing
	p.mu.Unlock()

	// Process pending orders
	updatedOrders := p.processOrders(candle, ordersToProcess)

	// Re-acquire lock to update order status
	p.mu.Lock()
	p.orders = updatedOrders

	// Update portfolio values if the candle is complete
	if candle.Complete {
		p.updatePortfolioValues(candle)
	}
	p.mu.Unlock()
}

// processOrders processes pending orders based on the new candle
// This function doesn't hold the mutex lock
func (p *PaperWallet) processOrders(candle core.Candle, orders []core.Order) []core.Order {
	result := make([]core.Order, len(orders))
	copy(result, orders)

	// Acquire lock briefly to initialize volume if needed
	p.mu.Lock()
	if _, ok := p.volume[candle.Pair]; !ok {
		p.volume[candle.Pair] = 0
	}
	p.mu.Unlock()

	for i, order := range result {
		// Ignore orders that are not for this pair or that are not pending
		if order.Pair != candle.Pair || order.Status != core.OrderStatusTypeNew {
			continue
		}

		// Process the order based on side (buy/sell)
		if order.Side == core.SideTypeBuy {
			p.processBuyOrder(&result[i], candle)
		} else {
			p.processSellOrder(&result[i], &result, candle)
		}
	}

	return result
}

// processBuyOrder processes a buy order
// This function acquires the mutex when needed
func (p *PaperWallet) processBuyOrder(order *core.Order, candle core.Candle) {
	// Check if the buy price was reached
	if order.Price < candle.Close {
		return
	}

	asset, quote := SplitAssetQuote(order.Pair)

	p.mu.Lock()
	defer p.mu.Unlock()

	p.ensureAssetExists(asset)

	// Register volume
	p.volume[candle.Pair] += order.Price * order.Quantity

	// Update the order
	order.UpdatedAt = candle.Time
	order.Status = core.OrderStatusTypeFilled

	// Update average price and balances
	p.updateAveragePrice(order.Side, order.Pair, order.Quantity, order.Price)
	p.assets[asset].Free = p.assets[asset].Free + order.Quantity
	p.assets[quote].Lock = p.assets[quote].Lock - order.Price*order.Quantity
}

// processSellOrder processes a sell order
// This function acquires the mutex when needed
func (p *PaperWallet) processSellOrder(order *core.Order, orders *[]core.Order, candle core.Candle) {
	// Determine the execution price of the order
	var orderPrice float64

	// Check order type and if the price was reached
	if isLimitOrder(order.Type) && candle.High >= order.Price {
		orderPrice = order.Price
	} else if isStopOrder(order.Type) && order.Stop != nil && candle.Low <= *order.Stop {
		orderPrice = *order.Stop
	} else {
		return // Price not reached
	}

	asset, quote := SplitAssetQuote(order.Pair)

	p.mu.Lock()
	defer p.mu.Unlock()

	p.ensureAssetExists(quote)

	// Cancel other orders from the same group
	if order.GroupID != nil {
		p.cancelRelatedOrdersLocked(order, *orders, candle.Time)
	}

	// Register volume
	orderVolume := order.Quantity * orderPrice
	p.volume[candle.Pair] += orderVolume

	// Update the order
	order.UpdatedAt = candle.Time
	order.Status = core.OrderStatusTypeFilled

	// Update average price and balances
	p.updateAveragePrice(order.Side, order.Pair, order.Quantity, orderPrice)
	p.assets[asset].Lock = p.assets[asset].Lock - order.Quantity
	p.assets[quote].Free = p.assets[quote].Free + order.Quantity*orderPrice
}

// isLimitOrder checks if it's a limit order type
func isLimitOrder(orderType core.OrderType) bool {
	return orderType == core.OrderTypeLimit ||
		orderType == core.OrderTypeLimitMaker ||
		orderType == core.OrderTypeTakeProfit ||
		orderType == core.OrderTypeTakeProfitLimit
}

// isStopOrder checks if it's a stop order type
func isStopOrder(orderType core.OrderType) bool {
	return orderType == core.OrderTypeStopLossLimit ||
		orderType == core.OrderTypeStopLoss
}

// cancelRelatedOrdersLocked cancels other orders from the same group
// This function assumes the mutex is already locked
func (p *PaperWallet) cancelRelatedOrdersLocked(order *core.Order, orders []core.Order, timestamp time.Time) {
	for j, groupOrder := range orders {
		if groupOrder.GroupID != nil && *groupOrder.GroupID == *order.GroupID &&
			groupOrder.ExchangeID != order.ExchangeID {
			orders[j].Status = core.OrderStatusTypeCanceled
			orders[j].UpdatedAt = timestamp
			break
		}
	}
}

// updatePortfolioValues updates the wallet values
// This function assumes the mutex is already locked
func (p *PaperWallet) updatePortfolioValues(candle core.Candle) {
	var total float64

	// Calculate the total value of each asset
	for asset, info := range p.assets {
		amount := info.Free + info.Lock
		pair := strings.ToUpper(asset + p.baseCoin)

		// Calculate asset value
		var assetValue float64
		if amount < 0 {
			v := math.Abs(amount)
			liquid := 2*v*p.avgShortPrice[pair] - v*p.lastCandle[pair].Close
			total += liquid
			assetValue = liquid
		} else {
			assetValue = amount * p.lastCandle[pair].Close
			total += assetValue
		}

		// Register asset value
		p.assetValues[asset] = append(p.assetValues[asset], AssetValue{
			Time:  candle.Time,
			Value: assetValue,
		})
	}

	// Register total wallet value
	baseCoinInfo := p.assets[p.baseCoin]
	p.equityValues = append(p.equityValues, AssetValue{
		Time:  candle.Time,
		Value: total + baseCoinInfo.Lock + baseCoinInfo.Free,
	})
}

// ---------------------
// Account Management
// ---------------------

// Account returns account information
func (p *PaperWallet) Account(_ context.Context) (core.Account, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	balances := make([]core.Balance, 0, len(p.assets))
	for pair, info := range p.assets {
		balances = append(balances, core.Balance{
			Asset: pair,
			Free:  info.Free,
			Lock:  info.Lock,
		})
	}

	return core.NewAccount(balances)
}

// Position returns the position of a pair
func (p *PaperWallet) Position(ctx context.Context, pair string) (asset, quote float64, err error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	assetTick, quoteTick := SplitAssetQuote(pair)
	acc, err := p.Account(ctx)
	if err != nil {
		return 0, 0, err
	}

	assetBalance, quoteBalance := acc.GetBalance(assetTick, quoteTick)
	return assetBalance.Free + assetBalance.Lock, quoteBalance.Free + quoteBalance.Lock, nil
}

// ---------------------
// Order Management
// ---------------------

// CreateOrderOCO creates an OCO (One-Cancels-the-Other) order
func (p *PaperWallet) CreateOrderOCO(_ context.Context, side core.SideType, pair string,
	size, price, stop, stopLimit float64) ([]core.Order, error) {
	if size == 0 {
		return nil, ErrInvalidQuantity
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Check available funds
	err := p.validateFunds(side, pair, size, price, false)
	if err != nil {
		return nil, err
	}

	// Create group ID for orders
	groupID := p.ID()

	// Create limit order
	limitMaker := core.Order{
		ExchangeID: p.ID(),
		CreatedAt:  p.lastCandle[pair].Time,
		UpdatedAt:  p.lastCandle[pair].Time,
		Pair:       pair,
		Side:       side,
		Type:       core.OrderTypeLimitMaker,
		Status:     core.OrderStatusTypeNew,
		Price:      price,
		Quantity:   size,
		GroupID:    &groupID,
		RefPrice:   p.lastCandle[pair].Close,
	}

	// Create stop order
	stopOrder := core.Order{
		ExchangeID: p.ID(),
		CreatedAt:  p.lastCandle[pair].Time,
		UpdatedAt:  p.lastCandle[pair].Time,
		Pair:       pair,
		Side:       side,
		Type:       core.OrderTypeStopLoss,
		Status:     core.OrderStatusTypeNew,
		Price:      stopLimit,
		Stop:       &stop,
		Quantity:   size,
		GroupID:    &groupID,
		RefPrice:   p.lastCandle[pair].Close,
	}

	// Add orders to the list
	p.orders = append(p.orders, limitMaker, stopOrder)

	return []core.Order{limitMaker, stopOrder}, nil
}

// CreateOrderLimit creates a limit order
func (p *PaperWallet) CreateOrderLimit(_ context.Context, side core.SideType, pair string,
	size float64, limit float64) (core.Order, error) {
	if size == 0 {
		return core.Order{}, ErrInvalidQuantity
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Check available funds
	err := p.validateFunds(side, pair, size, limit, false)
	if err != nil {
		return core.Order{}, err
	}

	// Create order
	order := core.Order{
		ExchangeID: p.ID(),
		CreatedAt:  p.lastCandle[pair].Time,
		UpdatedAt:  p.lastCandle[pair].Time,
		Pair:       pair,
		Side:       side,
		Type:       core.OrderTypeLimit,
		Status:     core.OrderStatusTypeNew,
		Price:      limit,
		Quantity:   size,
	}

	// Add order to the list
	p.orders = append(p.orders, order)

	return order, nil
}

// CreateOrderMarket creates a market order
func (p *PaperWallet) CreateOrderMarket(ctx context.Context, side core.SideType, pair string, size float64) (core.Order, error) {
	if size == 0 {
		return core.Order{}, ErrInvalidQuantity
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Check and apply funds (with immediate fill)
	err := p.validateFunds(side, pair, size, p.lastCandle[pair].Close, true)
	if err != nil {
		return core.Order{}, err
	}

	// Initialize volume if needed
	if _, ok := p.volume[pair]; !ok {
		p.volume[pair] = 0
	}

	// Register volume
	p.volume[pair] += p.lastCandle[pair].Close * size

	// Create order (already filled)
	order := core.Order{
		ExchangeID: p.ID(),
		CreatedAt:  p.lastCandle[pair].Time,
		UpdatedAt:  p.lastCandle[pair].Time,
		Pair:       pair,
		Side:       side,
		Type:       core.OrderTypeMarket,
		Status:     core.OrderStatusTypeFilled,
		Price:      p.lastCandle[pair].Close,
		Quantity:   size,
	}

	// Add order to the list
	p.orders = append(p.orders, order)

	return order, nil
}

// CreateOrderStop creates a stop order
func (p *PaperWallet) CreateOrderStop(_ context.Context, pair string, size float64, limit float64) (core.Order, error) {
	if size == 0 {
		return core.Order{}, ErrInvalidQuantity
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Check available funds
	err := p.validateFunds(core.SideTypeSell, pair, size, limit, false)
	if err != nil {
		return core.Order{}, err
	}

	// Create order
	order := core.Order{
		ExchangeID: p.ID(),
		CreatedAt:  p.lastCandle[pair].Time,
		UpdatedAt:  p.lastCandle[pair].Time,
		Pair:       pair,
		Side:       core.SideTypeSell,
		Type:       core.OrderTypeStopLossLimit,
		Status:     core.OrderStatusTypeNew,
		Price:      limit,
		Stop:       &limit,
		Quantity:   size,
	}

	// Add order to the list
	p.orders = append(p.orders, order)

	return order, nil
}

// CreateOrderMarketQuote creates a market order with a quantity in quote currency
func (p *PaperWallet) CreateOrderMarketQuote(
	ctx context.Context,
	side core.SideType,
	pair string,
	quoteQuantity float64,
) (core.Order, error) {
	p.mu.Lock()

	// Convert the quantity in quote currency to asset quantity
	info, err := p.AssetsInfo(pair)
	if err != nil {
		return core.Order{}, err
	}

	price := p.lastCandle[pair].Close
	quantity := common.AmountToLotSize(info.StepSize, info.BaseAssetPrecision, quoteQuantity/price)

	// Unlock before calling CreateOrderMarket to avoid deadlock
	p.mu.Unlock()

	return p.CreateOrderMarket(ctx, side, pair, quantity)
}

// Cancel cancels an order
func (p *PaperWallet) Cancel(_ context.Context, order core.Order) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, o := range p.orders {
		if o.ExchangeID == order.ExchangeID {
			// Mark order as canceled
			p.orders[i].Status = core.OrderStatusTypeCanceled

			// Release locked funds
			asset, quote := SplitAssetQuote(o.Pair)

			// Case 1: We have a long position and this is a sell order
			if p.assets[asset].Lock > 0 && o.Side == core.SideTypeSell {
				p.assets[asset].Free += o.Quantity
				p.assets[asset].Lock -= o.Quantity
			} else if p.assets[asset].Lock == 0 {
				// Case 2: We don't have a long position
				amount := order.Price * order.Quantity
				p.assets[quote].Free += amount
				p.assets[quote].Lock -= amount
			}

			return nil
		}
	}

	return errors.New("order not found")
}

// Order returns a specific order
func (p *PaperWallet) Order(_ context.Context, _ string, id int64) (core.Order, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, order := range p.orders {
		if order.ExchangeID == id {
			order.ID = id
			return order, nil
		}
	}

	return core.Order{}, errors.New("order not found")
}

// ---------------------
// Data Feed Methods
// ---------------------

// CandlesByPeriod returns candles within a period
func (p *PaperWallet) CandlesByPeriod(ctx context.Context, pair, period string,
	start, end time.Time) ([]core.Candle, error) {
	return p.feeder.CandlesByPeriod(ctx, pair, period, start, end)
}

// CandlesByLimit returns a limited number of candles
func (p *PaperWallet) CandlesByLimit(ctx context.Context, pair, period string, limit int) ([]core.Candle, error) {
	return p.feeder.CandlesByLimit(ctx, pair, period, limit)
}

// CandlesSubscription returns a channel to receive candles
func (p *PaperWallet) CandlesSubscription(ctx context.Context, pair, timeframe string) (chan core.Candle, chan error) {
	return p.feeder.CandlesSubscription(ctx, pair, timeframe)
}
