package exchange

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/raykavin/backnrun/internal/core"

	"github.com/adshao/go-binance/v2/common"

	"github.com/rodrigo-brito/ninjabot/tools/log"
)

// AssetValue representa o valor de um ativo em um momento específico
type AssetValue struct {
	Time  time.Time
	Value float64
}

// assetInfo representa a informação de saldo de um ativo
type assetInfo struct {
	Free float64
	Lock float64
}

// PaperWallet implementa uma carteira simulada para backtesting
type PaperWallet struct {
	mu sync.RWMutex

	// Contexto e configurações
	ctx          context.Context
	baseCoin     string
	counter      int64
	takerFee     float64
	makerFee     float64
	initialValue float64
	feeder       core.Feeder

	// Dados da carteira
	orders        []core.Order
	assets        map[string]*assetInfo
	avgShortPrice map[string]float64
	avgLongPrice  map[string]float64
	volume        map[string]float64

	// Dados de candles
	lastCandle map[string]core.Candle
	fistCandle map[string]core.Candle

	// Histórico de valores
	assetValues  map[string][]AssetValue
	equityValues []AssetValue
}

// PaperWalletOption define uma função de opção para configurar PaperWallet
type PaperWalletOption func(*PaperWallet)

// WithPaperAsset adiciona um ativo inicial à carteira
func WithPaperAsset(pair string, amount float64) PaperWalletOption {
	return func(wallet *PaperWallet) {
		wallet.assets[pair] = &assetInfo{
			Free: amount,
			Lock: 0,
		}
	}
}

// WithPaperFee configura as taxas da carteira
func WithPaperFee(maker, taker float64) PaperWalletOption {
	return func(wallet *PaperWallet) {
		wallet.makerFee = maker
		wallet.takerFee = taker
	}
}

// WithDataFeed configura o provedor de dados
func WithDataFeed(feeder core.Feeder) PaperWalletOption {
	return func(wallet *PaperWallet) {
		wallet.feeder = feeder
	}
}

// NewPaperWallet cria uma nova carteira simulada
func NewPaperWallet(ctx context.Context, baseCoin string, options ...PaperWalletOption) *PaperWallet {
	wallet := PaperWallet{
		ctx:           ctx,
		baseCoin:      baseCoin,
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

	// Aplica as opções
	for _, option := range options {
		option(&wallet)
	}

	// Inicializa o valor inicial da carteira
	wallet.initialValue = wallet.getAssetFreeAmount(wallet.baseCoin)

	log.Info("[SETUP] Using paper wallet")
	log.Infof("[SETUP] Initial Portfolio = %f %s", wallet.initialValue, wallet.baseCoin)

	return &wallet
}

// ID gera um ID único para ordens
func (p *PaperWallet) ID() int64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.counter++
	return p.counter
}

// AssetsInfo retorna informações sobre os ativos de um par
func (p *PaperWallet) AssetsInfo(pair string) core.AssetInfo {
	asset, quote := SplitAssetQuote(pair)
	return core.AssetInfo{
		BaseAsset:          asset,
		QuoteAsset:         quote,
		MaxPrice:           math.MaxFloat64,
		MaxQuantity:        math.MaxFloat64,
		StepSize:           0.00000001,
		TickSize:           0.00000001,
		QuotePrecision:     8,
		BaseAssetPrecision: 8,
	}
}

// Pairs retorna a lista de pares disponíveis na carteira
func (p *PaperWallet) Pairs() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	pairs := make([]string, 0, len(p.assets))
	for pair := range p.assets {
		pairs = append(pairs, pair)
	}
	return pairs
}

// LastQuote retorna a última cotação de um par
func (p *PaperWallet) LastQuote(ctx context.Context, pair string) (float64, error) {
	return p.feeder.LastQuote(ctx, pair)
}

// AssetValues retorna o histórico de valores de um ativo
func (p *PaperWallet) AssetValues(pair string) []AssetValue {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.assetValues[pair]
}

// EquityValues retorna o histórico de valores da carteira
func (p *PaperWallet) EquityValues() []AssetValue {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.equityValues
}

// MaxDrawdown calcula o drawdown máximo da carteira
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

// Summary imprime um resumo da carteira
func (p *PaperWallet) Summary() {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var (
		total        float64
		marketChange float64
		volume       float64
	)

	fmt.Println("----- FINAL WALLET -----")

	// Calcula o valor total dos ativos
	for pair := range p.lastCandle {
		asset, quote := SplitAssetQuote(pair)
		assetInfo, ok := p.assets[asset]
		if !ok {
			continue
		}

		quantity := assetInfo.Free + assetInfo.Lock

		// Calcula o valor do ativo
		value := p.calculateAssetValue(pair, asset, quantity)
		total += value

		// Calcula a variação do mercado
		marketChange += p.calculateMarketChange(pair)

		fmt.Printf("%.4f %s = %.4f %s\n", quantity, asset, value, quote)
	}

	// Calcula a variação média do mercado
	avgMarketChange := marketChange / float64(len(p.lastCandle))

	// Calcula o saldo na moeda base
	baseCoinValue := p.getAssetTotalAmount(p.baseCoin)

	// Calcula o lucro
	profit := total + baseCoinValue - p.initialValue

	// Imprime informações da moeda base
	fmt.Printf("%.4f %s\n", baseCoinValue, p.baseCoin)
	fmt.Println()

	// Calcula o drawdown máximo
	maxDrawDown, _, _ := p.MaxDrawdown()

	// Imprime o resumo de retornos
	fmt.Println("----- RETURNS -----")
	fmt.Printf("START PORTFOLIO     = %.2f %s\n", p.initialValue, p.baseCoin)
	fmt.Printf("FINAL PORTFOLIO     = %.2f %s\n", total+baseCoinValue, p.baseCoin)
	fmt.Printf("GROSS PROFIT        =  %f %s (%.2f%%)\n", profit, p.baseCoin, profit/p.initialValue*100)
	fmt.Printf("MARKET CHANGE (B&H) =  %.2f%%\n", avgMarketChange*100)
	fmt.Println()

	// Imprime informações de risco
	fmt.Println("------ RISK -------")
	fmt.Printf("MAX DRAWDOWN = %.2f %%\n", maxDrawDown*100)
	fmt.Println()

	// Imprime informações de volume
	fmt.Println("------ VOLUME -----")
	for pair, vol := range p.volume {
		volume += vol
		fmt.Printf("%s         = %.2f %s\n", pair, vol, p.baseCoin)
	}
	fmt.Printf("TOTAL           = %.2f %s\n", volume, p.baseCoin)
	fmt.Println("-------------------")
}

// calculateAssetValue calcula o valor de um ativo
func (p *PaperWallet) calculateAssetValue(pair, asset string, quantity float64) float64 {
	if quantity == 0 {
		return 0
	}

	// Se a quantidade for positiva, é uma posição longa
	if quantity > 0 {
		return quantity * p.lastCandle[pair].Close
	}

	// Se a quantidade for negativa, é uma posição curta
	// Calcula o valor total da posição curta
	totalShort := 2.0*p.avgShortPrice[pair]*quantity - p.lastCandle[pair].Close*quantity
	return math.Abs(totalShort)
}

// calculateMarketChange calcula a variação do preço de um par
func (p *PaperWallet) calculateMarketChange(pair string) float64 {
	firstPrice := p.fistCandle[pair].Close
	lastPrice := p.lastCandle[pair].Close
	return (lastPrice - firstPrice) / firstPrice
}

// getAssetFreeAmount retorna o saldo livre de um ativo
func (p *PaperWallet) getAssetFreeAmount(asset string) float64 {
	assetInfo, ok := p.assets[asset]
	if !ok {
		return 0
	}
	return assetInfo.Free
}

// getAssetTotalAmount retorna o saldo total (livre + bloqueado) de um ativo
func (p *PaperWallet) getAssetTotalAmount(asset string) float64 {
	assetInfo, ok := p.assets[asset]
	if !ok {
		return 0
	}
	return assetInfo.Free + assetInfo.Lock
}

// ensureAssetExists garante que um ativo existe na carteira
func (p *PaperWallet) ensureAssetExists(asset string) {
	if _, ok := p.assets[asset]; !ok {
		p.assets[asset] = &assetInfo{}
	}
}

// validateFunds verifica se há fundos suficientes para uma operação
func (p *PaperWallet) validateFunds(side core.SideType, pair string, amount, value float64, fill bool) error {
	asset, quote := SplitAssetQuote(pair)

	// Garante que os ativos existem
	p.ensureAssetExists(asset)
	p.ensureAssetExists(quote)

	// Verifica se há fundos suficientes para a operação
	if side == core.SideTypeSell {
		return p.validateSellFunds(pair, asset, quote, amount, value, fill)
	} else { // SideTypeBuy
		return p.validateBuyFunds(pair, asset, quote, amount, value, fill)
	}
}

// validateSellFunds verifica e processa fundos para venda
func (p *PaperWallet) validateSellFunds(pair, asset, quote string, amount, value float64, fill bool) error {
	// Calcula os fundos disponíveis
	funds := p.assets[quote].Free
	if p.assets[asset].Free > 0 {
		funds += p.assets[asset].Free * value
	}

	// Verifica se há fundos suficientes
	if funds < amount*value {
		return &OrderError{
			Err:      ErrInsufficientFunds,
			Pair:     pair,
			Quantity: amount,
		}
	}

	// Calcula os valores a serem bloqueados
	lockedAsset := math.Min(math.Max(p.assets[asset].Free, 0), amount) // ignora valores negativos
	lockedQuote := (amount - lockedAsset) * value

	// Atualiza os saldos
	p.assets[asset].Free -= lockedAsset
	p.assets[quote].Free -= lockedQuote

	if fill {
		// Atualiza o preço médio
		p.updateAveragePrice(core.SideTypeSell, pair, amount, value)

		if lockedQuote > 0 { // entrando em posição curta
			p.assets[asset].Free -= amount
		} else { // liquidando posição longa
			p.assets[quote].Free += amount * value
		}
	} else {
		// Bloqueia os valores
		p.assets[asset].Lock += lockedAsset
		p.assets[quote].Lock += lockedQuote
	}

	log.Debugf("%s -> LOCK = %f / FREE %f", asset, p.assets[asset].Lock, p.assets[asset].Free)
	return nil
}

// validateBuyFunds verifica e processa fundos para compra
func (p *PaperWallet) validateBuyFunds(pair, asset, quote string, amount, value float64, fill bool) error {
	var liquidShortValue float64

	// Se há posição curta, calcula o valor de liquidação
	if p.assets[asset].Free < 0 {
		v := math.Abs(p.assets[asset].Free)
		liquidShortValue = 2*v*p.avgShortPrice[pair] - v*value
		funds := p.assets[quote].Free + liquidShortValue

		// Calcula a quantidade efetiva a comprar
		amountToBuy := amount
		if p.assets[asset].Free < 0 {
			amountToBuy = amount + p.assets[asset].Free
		}

		// Verifica se há fundos suficientes
		if funds < amountToBuy*value {
			return &OrderError{
				Err:      ErrInsufficientFunds,
				Pair:     pair,
				Quantity: amount,
			}
		}

		// Calcula os valores a serem bloqueados
		lockedAsset := math.Min(-math.Min(p.assets[asset].Free, 0), amount)
		lockedQuote := (amount-lockedAsset)*value - liquidShortValue

		// Atualiza os saldos
		p.assets[asset].Free += lockedAsset
		p.assets[quote].Free -= lockedQuote

		if fill {
			// Atualiza o preço médio
			p.updateAveragePrice(core.SideTypeBuy, pair, amount, value)
			p.assets[asset].Free += amount - lockedAsset
		} else {
			// Bloqueia os valores
			p.assets[asset].Lock += lockedAsset
			p.assets[quote].Lock += lockedQuote
		}

		log.Debugf("%s -> LOCK = %f / FREE %f", asset, p.assets[asset].Lock, p.assets[asset].Free)
	} else {
		// Caso simples: compra com saldo em quote
		if p.assets[quote].Free < amount*value {
			return &OrderError{
				Err:      ErrInsufficientFunds,
				Pair:     pair,
				Quantity: amount,
			}
		}

		if fill {
			// Atualiza o preço médio e saldos diretamente
			p.updateAveragePrice(core.SideTypeBuy, pair, amount, value)
			p.assets[quote].Free -= amount * value
			p.assets[asset].Free += amount
		} else {
			// Bloqueia os valores
			p.assets[quote].Lock += amount * value
			p.assets[quote].Free -= amount * value
		}
	}

	return nil
}

// updateAveragePrice atualiza o preço médio de compra/venda
func (p *PaperWallet) updateAveragePrice(side core.SideType, pair string, amount, value float64) {
	actualQty := 0.0
	asset, quote := SplitAssetQuote(pair)

	if p.assets[asset] != nil {
		actualQty = p.assets[asset].Free
	}

	// Sem posição prévia
	if actualQty == 0 {
		if side == core.SideTypeBuy {
			p.avgLongPrice[pair] = value
		} else {
			p.avgShortPrice[pair] = value
		}
		return
	}

	// Posição longa + ordem de compra
	if actualQty > 0 && side == core.SideTypeBuy {
		positionValue := p.avgLongPrice[pair] * actualQty
		p.avgLongPrice[pair] = (positionValue + amount*value) / (actualQty + amount)
		return
	}

	// Posição longa + ordem de venda
	if actualQty > 0 && side == core.SideTypeSell {
		// Calcula o lucro
		profitValue := amount*value - math.Min(amount, actualQty)*p.avgLongPrice[pair]
		percentage := profitValue / (amount * p.avgLongPrice[pair])
		log.Infof("PROFIT = %.4f %s (%.2f %%)", profitValue, quote, percentage*100.0)

		// Se a quantidade vendida não fecha a posição
		if amount <= actualQty {
			return
		}

		// Se a venda excede a posição, inicia posição curta
		p.avgShortPrice[pair] = value
		return
	}

	// Posição curta + ordem de venda
	if actualQty < 0 && side == core.SideTypeSell {
		positionValue := p.avgShortPrice[pair] * -actualQty
		p.avgShortPrice[pair] = (positionValue + amount*value) / (-actualQty + amount)
		return
	}

	// Posição curta + ordem de compra
	if actualQty < 0 && side == core.SideTypeBuy {
		// Calcula o lucro
		profitValue := math.Min(amount, -actualQty)*p.avgShortPrice[pair] - amount*value
		percentage := profitValue / (amount * p.avgShortPrice[pair])
		log.Infof("PROFIT = %.4f %s (%.2f %%)", profitValue, quote, percentage*100.0)

		// Se a quantidade comprada não fecha a posição
		if amount <= -actualQty {
			return
		}

		// Se a compra excede a posição curta, inicia posição longa
		p.avgLongPrice[pair] = value
	}
}

// OnCandle processa um novo candle
func (p *PaperWallet) OnCandle(candle core.Candle) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Atualiza o candle mais recente
	p.lastCandle[candle.Pair] = candle

	// Registra o primeiro candle, se ainda não existir
	if _, ok := p.fistCandle[candle.Pair]; !ok {
		p.fistCandle[candle.Pair] = candle
	}

	// Processa ordens pendentes
	p.processOrders(candle)

	// Atualiza valores da carteira se o candle estiver completo
	if candle.Complete {
		p.updatePortfolioValues(candle)
	}
}

// processOrders processa as ordens pendentes com base no novo candle
func (p *PaperWallet) processOrders(candle core.Candle) {
	for i, order := range p.orders {
		// Ignora ordens que não são para este par ou que não estão pendentes
		if order.Pair != candle.Pair || order.Status != core.OrderStatusTypeNew {
			continue
		}

		// Inicializa o volume para o par, se necessário
		if _, ok := p.volume[candle.Pair]; !ok {
			p.volume[candle.Pair] = 0
		}

		// Processa a ordem com base no lado (compra/venda)
		if order.Side == core.SideTypeBuy {
			p.processBuyOrder(i, order, candle)
		} else {
			p.processSellOrder(i, order, candle)
		}
	}
}

// processBuyOrder processa uma ordem de compra
func (p *PaperWallet) processBuyOrder(orderIndex int, order core.Order, candle core.Candle) {
	// Verifica se o preço de compra foi atingido
	if order.Price < candle.Close {
		return
	}

	asset, quote := SplitAssetQuote(order.Pair)
	p.ensureAssetExists(asset)

	// Registra o volume
	p.volume[candle.Pair] += order.Price * order.Quantity

	// Atualiza a ordem
	p.orders[orderIndex].UpdatedAt = candle.Time
	p.orders[orderIndex].Status = core.OrderStatusTypeFilled

	// Atualiza o preço médio e os saldos
	p.updateAveragePrice(order.Side, order.Pair, order.Quantity, order.Price)
	p.assets[asset].Free = p.assets[asset].Free + order.Quantity
	p.assets[quote].Lock = p.assets[quote].Lock - order.Price*order.Quantity
}

// processSellOrder processa uma ordem de venda
func (p *PaperWallet) processSellOrder(orderIndex int, order core.Order, candle core.Candle) {
	// Determina o preço de execução da ordem
	var orderPrice float64

	// Verifica o tipo de ordem e se o preço foi atingido
	if isLimitOrder(order.Type) && candle.High >= order.Price {
		orderPrice = order.Price
	} else if isStopOrder(order.Type) && order.Stop != nil && candle.Low <= *order.Stop {
		orderPrice = *order.Stop
	} else {
		return // Preço não atingido
	}

	// Cancela outras ordens do mesmo grupo
	if order.GroupID != nil {
		p.cancelRelatedOrders(order, candle.Time)
	}

	asset, quote := SplitAssetQuote(order.Pair)
	p.ensureAssetExists(quote)

	// Registra o volume
	orderVolume := order.Quantity * orderPrice
	p.volume[candle.Pair] += orderVolume

	// Atualiza a ordem
	p.orders[orderIndex].UpdatedAt = candle.Time
	p.orders[orderIndex].Status = core.OrderStatusTypeFilled

	// Atualiza o preço médio e os saldos
	p.updateAveragePrice(order.Side, order.Pair, order.Quantity, orderPrice)
	p.assets[asset].Lock = p.assets[asset].Lock - order.Quantity
	p.assets[quote].Free = p.assets[quote].Free + order.Quantity*orderPrice
}

// isLimitOrder verifica se é uma ordem do tipo limite
func isLimitOrder(orderType core.OrderType) bool {
	return orderType == core.OrderTypeLimit ||
		orderType == core.OrderTypeLimitMaker ||
		orderType == core.OrderTypeTakeProfit ||
		orderType == core.OrderTypeTakeProfitLimit
}

// isStopOrder verifica se é uma ordem do tipo stop
func isStopOrder(orderType core.OrderType) bool {
	return orderType == core.OrderTypeStopLossLimit ||
		orderType == core.OrderTypeStopLoss
}

// cancelRelatedOrders cancela outras ordens do mesmo grupo
func (p *PaperWallet) cancelRelatedOrders(order core.Order, timestamp time.Time) {
	for j, groupOrder := range p.orders {
		if groupOrder.GroupID != nil && *groupOrder.GroupID == *order.GroupID &&
			groupOrder.ExchangeID != order.ExchangeID {
			p.orders[j].Status = core.OrderStatusTypeCanceled
			p.orders[j].UpdatedAt = timestamp
			break
		}
	}
}

// updatePortfolioValues atualiza os valores da carteira
func (p *PaperWallet) updatePortfolioValues(candle core.Candle) {
	var total float64

	// Calcula o valor total de cada ativo
	for asset, info := range p.assets {
		amount := info.Free + info.Lock
		pair := strings.ToUpper(asset + p.baseCoin)

		// Calcula o valor do ativo
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

		// Registra o valor do ativo
		p.assetValues[asset] = append(p.assetValues[asset], AssetValue{
			Time:  candle.Time,
			Value: assetValue,
		})
	}

	// Registra o valor total da carteira
	baseCoinInfo := p.assets[p.baseCoin]
	p.equityValues = append(p.equityValues, AssetValue{
		Time:  candle.Time,
		Value: total + baseCoinInfo.Lock + baseCoinInfo.Free,
	})
}

// Account retorna informações da conta
func (p *PaperWallet) Account() (core.Account, error) {
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

	return core.Account{
		Balances: balances,
	}, nil
}

// Position retorna a posição de um par
func (p *PaperWallet) Position(pair string) (asset, quote float64, err error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	assetTick, quoteTick := SplitAssetQuote(pair)
	acc, err := p.Account()
	if err != nil {
		return 0, 0, err
	}

	assetBalance, quoteBalance := acc.GetBalance(assetTick, quoteTick)
	return assetBalance.Free + assetBalance.Lock, quoteBalance.Free + quoteBalance.Lock, nil
}

// CreateOrderOCO cria uma ordem OCO (One-Cancels-the-Other)
func (p *PaperWallet) CreateOrderOCO(side core.SideType, pair string,
	size, price, stop, stopLimit float64) ([]core.Order, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if size == 0 {
		return nil, ErrInvalidQuantity
	}

	// Verifica fundos disponíveis
	err := p.validateFunds(side, pair, size, price, false)
	if err != nil {
		return nil, err
	}

	// Cria ID de grupo para as ordens
	groupID := p.ID()

	// Cria a ordem de limite
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

	// Cria a ordem de stop
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

	// Adiciona as ordens à lista
	p.orders = append(p.orders, limitMaker, stopOrder)

	return []core.Order{limitMaker, stopOrder}, nil
}

// CreateOrderLimit cria uma ordem de limite
func (p *PaperWallet) CreateOrderLimit(side core.SideType, pair string,
	size float64, limit float64) (core.Order, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if size == 0 {
		return core.Order{}, ErrInvalidQuantity
	}

	// Verifica fundos disponíveis
	err := p.validateFunds(side, pair, size, limit, false)
	if err != nil {
		return core.Order{}, err
	}

	// Cria a ordem
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

	// Adiciona a ordem à lista
	p.orders = append(p.orders, order)

	return order, nil
}

// CreateOrderMarket cria uma ordem de mercado
func (p *PaperWallet) CreateOrderMarket(side core.SideType, pair string, size float64) (core.Order, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.createOrderMarket(side, pair, size)
}

// CreateOrderStop cria uma ordem de stop
func (p *PaperWallet) CreateOrderStop(pair string, size float64, limit float64) (core.Order, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if size == 0 {
		return core.Order{}, ErrInvalidQuantity
	}

	// Verifica fundos disponíveis
	err := p.validateFunds(core.SideTypeSell, pair, size, limit, false)
	if err != nil {
		return core.Order{}, err
	}

	// Cria a ordem
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

	// Adiciona a ordem à lista
	p.orders = append(p.orders, order)

	return order, nil
}

// createOrderMarket é um método interno para criar ordens de mercado
func (p *PaperWallet) createOrderMarket(side core.SideType, pair string, size float64) (core.Order, error) {
	if size == 0 {
		return core.Order{}, ErrInvalidQuantity
	}

	// Verifica e aplica os fundos (com preenchimento imediato)
	err := p.validateFunds(side, pair, size, p.lastCandle[pair].Close, true)
	if err != nil {
		return core.Order{}, err
	}

	// Inicializa o volume, se necessário
	if _, ok := p.volume[pair]; !ok {
		p.volume[pair] = 0
	}

	// Registra o volume
	p.volume[pair] += p.lastCandle[pair].Close * size

	// Cria a ordem (já preenchida)
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

	// Adiciona a ordem à lista
	p.orders = append(p.orders, order)

	return order, nil
}

// CreateOrderMarketQuote cria uma ordem de mercado com quantidade em moeda de cotação
func (p *PaperWallet) CreateOrderMarketQuote(side core.SideType, pair string,
	quoteQuantity float64) (core.Order, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Converte a quantidade em moeda de cotação para quantidade de ativo
	info := p.AssetsInfo(pair)
	quantity := common.AmountToLotSize(info.StepSize, info.BaseAssetPrecision, quoteQuantity/p.lastCandle[pair].Close)

	return p.createOrderMarket(side, pair, quantity)
}

// Cancel cancela uma ordem
func (p *PaperWallet) Cancel(order core.Order) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, o := range p.orders {
		if o.ExchangeID == order.ExchangeID {
			// Marca a ordem como cancelada
			p.orders[i].Status = core.OrderStatusTypeCanceled

			// Libera os fundos bloqueados
			asset, quote := SplitAssetQuote(o.Pair)

			// Caso 1: Temos posição longa e esta é uma ordem de venda
			if p.assets[asset].Lock > 0 && o.Side == core.SideTypeSell {
				p.assets[asset].Free += o.Quantity
				p.assets[asset].Lock -= o.Quantity
			} else if p.assets[asset].Lock == 0 {
				// Caso 2: Não temos posição longa
				amount := order.Price * order.Quantity
				p.assets[quote].Free += amount
				p.assets[quote].Lock -= amount
			}

			return nil
		}
	}

	return errors.New("order not found")
}

// Order retorna uma ordem específica
func (p *PaperWallet) Order(_ string, id int64) (core.Order, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, order := range p.orders {
		if order.ExchangeID == id {
			return order, nil
		}
	}

	return core.Order{}, errors.New("order not found")
}

// CandlesByPeriod retorna candles dentro de um período
func (p *PaperWallet) CandlesByPeriod(ctx context.Context, pair, period string,
	start, end time.Time) ([]core.Candle, error) {
	return p.feeder.CandlesByPeriod(ctx, pair, period, start, end)
}

// CandlesByLimit retorna um número limitado de candles
func (p *PaperWallet) CandlesByLimit(ctx context.Context, pair, period string, limit int) ([]core.Candle, error) {
	return p.feeder.CandlesByLimit(ctx, pair, period, limit)
}

// CandlesSubscription retorna um canal para receber candles
func (p *PaperWallet) CandlesSubscription(ctx context.Context, pair, timeframe string) (chan core.Candle, chan error) {
	return p.feeder.CandlesSubscription(ctx, pair, timeframe)
}
