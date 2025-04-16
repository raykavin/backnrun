package strategies

import (
	"github.com/markcheno/go-talib"
	"github.com/raykavin/backnrun"
	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/indicator"
)

type TrendMasterParameters struct{
	
}

// TrendMasterStrategy implementa uma estratégia que combina múltiplos indicadores
// para identificar tendências fortes e gerar sinais de entrada e saída
type TrendMasterStrategy struct {
	// Parâmetros dos indicadores
	emaFastPeriod    int
	emaSlowPeriod    int
	emaLongPeriod    int
	macdFast         int
	macdSlow         int
	macdSignal       int
	adxPeriod        int
	adxThreshold     float64
	profitTarget     float64
	stopLoss         float64
	positionSize     float64
	maxRiskPerTrade  float64
	trailStopPercent float64
	atrPeriod        int
	atrMultiplier    float64

	// Controle de trades
	consecutiveLosses int
	maxTradesPerDay   int
	dailyTradeCount   map[string]int
	lastTradeDate     string
	lastTradeResult   map[string]bool // true = ganho, false = perda

	// Filtros adicionais
	useRsiFilter  bool
	rsiPeriod     int
	rsiOverbought float64
	rsiOversold   float64
	useVolFilter  bool
	volAvgPeriod  int
	minVolRatio   float64

	// Rastreamento interno
	activeOrders map[string]int64   // Rastreia ordens stop ativas
	lastPrice    map[string]float64 // Rastreia o último preço para trailing stop
	entryPrice   map[string]float64 // Preço de entrada para cada par
}

// NewTrendMasterStrategy creates a new instance of the strategy with the default parameters
func NewTrendMasterStrategy() *TrendMasterStrategy {
	return &TrendMasterStrategy{
		emaFastPeriod:    9,
		emaSlowPeriod:    21,
		emaLongPeriod:    34,
		macdSlow:         150,
		macdFast:         14,
		macdSignal:       14,
		adxPeriod:        14,
		adxThreshold:     25.0,
		profitTarget:     0.06,
		stopLoss:         0.02,
		positionSize:     0.3,
		maxRiskPerTrade:  0.01,
		trailStopPercent: 0.03,
		atrPeriod:        14,
		atrMultiplier:    2.0,

		// Trade control
		consecutiveLosses: 0,
		maxTradesPerDay:   8,
		dailyTradeCount:   make(map[string]int),
		lastTradeResult:   make(map[string]bool),

		// Additional indicator filters
		useRsiFilter:  true,
		rsiPeriod:     14,
		rsiOverbought: 70.0,
		rsiOversold:   30.0,
		useVolFilter:  true,
		volAvgPeriod:  20,
		minVolRatio:   1.1,

		// Rastreamento interno
		activeOrders: make(map[string]int64),
		lastPrice:    make(map[string]float64),
		entryPrice:   make(map[string]float64),
	}
}

// Timeframe retorna o timeframe necessário para esta estratégia
func (t TrendMasterStrategy) Timeframe() string {
	return "15m"
}

// WarmupPeriod retorna o número de candles necessários antes da estratégia estar pronta
func (t TrendMasterStrategy) WarmupPeriod() int {
	// Usar o período do MACD lento + período de sinal + margem extra de segurança
	// O MACD precisa de muito mais dados históricos para ser calculado corretamente
	return t.macdSlow*2 + t.macdSignal + 100
}

// Indicators calcula e retorna os indicadores usados por esta estratégia
func (t TrendMasterStrategy) Indicators(df *core.Dataframe) []core.ChartIndicator {
	// Calcular EMAs
	df.Metadata["ema_fast"] = indicator.EMA(df.Close, t.emaFastPeriod)
	df.Metadata["ema_slow"] = indicator.EMA(df.Close, t.emaSlowPeriod)
	df.Metadata["ema_long_high"] = indicator.EMA(df.High, t.emaLongPeriod)
	df.Metadata["ema_long_low"] = indicator.EMA(df.Low, t.emaLongPeriod)

	// Calcular MACD
	df.Metadata["macd"], df.Metadata["macd_signal"], df.Metadata["macd_hist"] = indicator.MACD(
		df.Close,
		t.macdFast,
		t.macdSlow,
		t.macdSignal,
	)

	// Calcular ADX e directional indicators
	df.Metadata["adx"] = indicator.ADX(df.High, df.Low, df.Close, t.adxPeriod)
	df.Metadata["plus_di"] = talib.PlusDI(df.High, df.Low, df.Close, t.adxPeriod)
	df.Metadata["minus_di"] = talib.MinusDI(df.High, df.Low, df.Close, t.adxPeriod)

	// Adicionar ATR para cálculo de volatilidade
	df.Metadata["atr"] = talib.Atr(df.High, df.Low, df.Close, t.atrPeriod)

	// Calcular RSI para filtro adicional
	if t.useRsiFilter {
		df.Metadata["rsi"] = talib.Rsi(df.Close, t.rsiPeriod)
	}

	// Calcular média de volume para filtro
	if t.useVolFilter {
		df.Metadata["vol_avg"] = indicator.SMA(df.Volume, t.volAvgPeriod)
	}

	// Retornar indicadores para visualização
	return []core.ChartIndicator{
		{
			Overlay:   true,
			GroupName: "Moving Averages",
			Time:      df.Time,
			Metrics: []core.IndicatorMetric{
				{
					Values: df.Metadata["ema_fast"],
					Name:   "EMA 9",
					Color:  "blue",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["ema_slow"],
					Name:   "EMA 21",
					Color:  "green",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["ema_long_high"],
					Name:   "EMA 34 (High)",
					Color:  "purple",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["ema_long_low"],
					Name:   "EMA 34 (Low)",
					Color:  "orange",
					Style:  core.StyleLine,
				},
			},
		},
		{
			Overlay:   false,
			GroupName: "MACD",
			Time:      df.Time,
			Metrics: []core.IndicatorMetric{
				{
					Values: df.Metadata["macd"],
					Name:   "MACD Line",
					Color:  "blue",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["macd_signal"],
					Name:   "Signal Line",
					Color:  "red",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["macd_hist"],
					Name:   "Histogram",
					Color:  "green",
					Style:  core.StyleHistogram,
				},
			},
		},
		{
			Overlay:   false,
			GroupName: "ADX",
			Time:      df.Time,
			Metrics: []core.IndicatorMetric{
				{
					Values: df.Metadata["adx"],
					Name:   "ADX",
					Color:  "black",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["plus_di"],
					Name:   "+DI",
					Color:  "green",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["minus_di"],
					Name:   "-DI",
					Color:  "red",
					Style:  core.StyleLine,
				},
			},
		},
		{
			Overlay:   false,
			GroupName: "ATR",
			Time:      df.Time,
			Metrics: []core.IndicatorMetric{
				{
					Values: df.Metadata["atr"],
					Name:   "ATR",
					Color:  "blue",
					Style:  core.StyleLine,
				},
			},
		},
	}
}

// OnCandle é chamado para cada novo candle e implementa a lógica de trading
func (t *TrendMasterStrategy) OnCandle(df *core.Dataframe, broker core.Broker) {
	pair := df.Pair
	closePrice := df.Close.Last(0)
	// Obter data atual (último candle)
	currentDate := ""
	if len(df.Time) > 0 {
		currentDate = df.Time[len(df.Time)-1].Format("2006-01-02")
	}

	// Resetar contagem diária de trades se estamos em um novo dia
	if currentDate != t.lastTradeDate {
		t.dailyTradeCount = make(map[string]int)
		t.lastTradeDate = currentDate
	}

	// Obter posição atual
	assetPosition, quotePosition, err := broker.Position(pair)
	if err != nil {
		backnrun.DefaultLog.Error(err)
		return
	}

	// Atualizar preço de trailing stop se em posição
	if assetPosition > 0 {
		lastPrice, exists := t.lastPrice[pair]
		if !exists || closePrice > lastPrice {
			t.lastPrice[pair] = closePrice
		}
	}

	// Verificar sinais de entrada e saída
	if assetPosition > 0 {
		// Já estamos em posição longa, verificar saída
		if t.shouldExit(df) || t.checkTrailingStop(df, pair) {
			t.executeExit(df, broker, assetPosition)
			// Reset após saída
			delete(t.lastPrice, pair)
		}
	} else {
		// Sem posição, verificar entrada (apenas se não excedemos o limite diário)
		tradeCount := t.dailyTradeCount[pair]
		if tradeCount < t.maxTradesPerDay && t.shouldEnter(df) {
			t.executeEntry(df, broker, quotePosition, closePrice)
			// Incrementar contador de trades
			t.dailyTradeCount[pair] = tradeCount + 1
		}
	}
}

// checkTrailingStop verifica se o trailing stop foi atingido
func (t *TrendMasterStrategy) checkTrailingStop(df *core.Dataframe, pair string) bool {
	closePrice := df.Close.Last(0)
	lastPrice, exists := t.lastPrice[pair]

	if !exists {
		return false
	}

	// Se o preço caiu abaixo do trailing stop, acionamos a saída
	trailAmount := lastPrice * t.trailStopPercent
	if closePrice <= lastPrice-trailAmount {
		backnrun.DefaultLog.WithFields(map[string]interface{}{
			"pair":         pair,
			"highestPrice": lastPrice,
			"currentPrice": closePrice,
			"trailAmount":  trailAmount,
		}).Info("Trailing stop ativado")
		return true
	}

	return false
}

// shouldEnter verifica se as condições de entrada são atendidas
func (t *TrendMasterStrategy) shouldEnter(df *core.Dataframe) bool {
	closePrice := df.Close.Last(0)
	emaLongHigh := df.Metadata["ema_long_high"].Last(0)
	emaFast := df.Metadata["ema_fast"].Last(0)
	emaSlow := df.Metadata["ema_slow"].Last(0)
	macd := df.Metadata["macd"].Last(0)
	macdSignal := df.Metadata["macd_signal"].Last(0)
	plusDI := df.Metadata["plus_di"].Last(0)
	minusDI := df.Metadata["minus_di"].Last(0)
	adx := df.Metadata["adx"].Last(0)
	atr := df.Metadata["atr"].Last(0)

	// Verificações básicas (condições originais)
	priceAboveEMA := closePrice > emaLongHigh
	macdAboveSignal := macd > macdSignal
	plusDIAboveMinusDI := plusDI > minusDI
	adxAboveThreshold := adx > t.adxThreshold
	emaFastAboveSlow := emaFast > emaSlow

	// Filtro de volatilidade: verificar se a volatilidade não está muito alta
	volatilityCheck := true
	if atr > 0 {
		volatilityRatio := atr / closePrice
		volatilityCheck = volatilityRatio < 0.015 // Volatilidade menor que 1.5%
	}

	// Filtro de RSI: evitar compras em mercado sobrecomprado
	rsiCheck := true
	if t.useRsiFilter {
		rsi := df.Metadata["rsi"].Last(0)
		rsiCheck = rsi < t.rsiOverbought // RSI não está sobrecomprado
	}

	// Filtro de volume: verificar se o volume é suficiente
	volumeCheck := true
	if t.useVolFilter {
		volume := df.Volume.Last(0)
		volumeAvg := df.Metadata["vol_avg"].Last(0)
		if volumeAvg > 0 {
			volumeRatio := volume / volumeAvg
			volumeCheck = volumeRatio >= t.minVolRatio // Volume acima da média
		}
	}

	// Verificar tendência de alta consistente
	bullishMarket := true // Simplificado para permitir mais sinais

	// Reduzir frequência de trades após perdas consecutivas
	riskAdjustment := true
	if t.consecutiveLosses >= 2 {
		// Após 2 perdas consecutivas, exigir condições mais rigorosas
		riskAdjustment = adx > (t.adxThreshold+5) && (plusDI-minusDI) > 10
	}

	// Todas as condições principais devem ser verdadeiras para entrada
	mainConditions := priceAboveEMA && macdAboveSignal && plusDIAboveMinusDI &&
		adxAboveThreshold && emaFastAboveSlow

	// Filtros adicionais para melhorar a qualidade dos sinais
	additionalFilters := volatilityCheck && rsiCheck && volumeCheck && bullishMarket && riskAdjustment

	return mainConditions && additionalFilters
}

// shouldExit verifica se as condições de saída são atendidas
func (t *TrendMasterStrategy) shouldExit(df *core.Dataframe) bool {
	closePrice := df.Close.Last(0)
	emaLongLow := df.Metadata["ema_long_low"].Last(0)
	emaFast := df.Metadata["ema_fast"].Last(0)
	emaSlow := df.Metadata["ema_slow"].Last(0)
	macd := df.Metadata["macd"].Last(0)
	macdSignal := df.Metadata["macd_signal"].Last(0)
	plusDI := df.Metadata["plus_di"].Last(0)
	minusDI := df.Metadata["minus_di"].Last(0)
	adx := df.Metadata["adx"].Last(0)

	// Condições de saída básicas
	priceBelowEMA := closePrice < emaLongLow
	macdBelowSignal := macd < macdSignal
	minusDIAbovePlusDI := minusDI > plusDI
	adxAboveThreshold := adx > t.adxThreshold
	emaFastBelowSlow := emaFast < emaSlow

	// Verificar a quebra da média móvel longa calculada na mínima
	// Este é um sinal de saída rápido independente das outras condições
	if priceBelowEMA {
		backnrun.DefaultLog.WithFields(map[string]interface{}{
			"pair":       df.Pair,
			"closePrice": closePrice,
			"emaLongLow": emaLongLow,
		}).Info("Saída rápida: preço abaixo da EMA Low")
		return true
	}

	// Saída rápida se o MACD cair muito rápido (potencial reversão forte)
	macdHist := df.Metadata["macd_hist"].Last(0)
	prevMacdHist := df.Metadata["macd_hist"].Last(1)
	if macdHist < 0 && prevMacdHist > 0 && macdHist < prevMacdHist*-1.5 {
		backnrun.DefaultLog.WithFields(map[string]interface{}{
			"pair":         df.Pair,
			"macdHist":     macdHist,
			"prevMacdHist": prevMacdHist,
		}).Info("Saída rápida: MACD revertendo fortemente")
		return true
	}

	// Saída rápida se o ADX começar a cair enquanto estamos em posição (tendência enfraquecendo)
	adxFalling := adx < df.Metadata["adx"].Last(1) && df.Metadata["adx"].Last(1) < df.Metadata["adx"].Last(2)
	if adxFalling && minusDIAbovePlusDI {
		backnrun.DefaultLog.WithFields(map[string]interface{}{
			"pair":    df.Pair,
			"adx":     adx,
			"prevAdx": df.Metadata["adx"].Last(1),
			"plusDI":  plusDI,
			"minusDI": minusDI,
		}).Info("Saída rápida: ADX caindo e -DI acima de +DI")
		return true
	}

	// RSI indicando sobrecompra extrema
	if t.useRsiFilter {
		rsi := df.Metadata["rsi"].Last(0)
		if rsi > 80 { // Sobrecompra extrema
			backnrun.DefaultLog.WithFields(map[string]interface{}{
				"pair": df.Pair,
				"rsi":  rsi,
			}).Info("Saída rápida: RSI em sobrecompra extrema")
			return true
		}
	}

	// Todas as condições regulares para saída
	regularExitConditions := macdBelowSignal && minusDIAbovePlusDI && adxAboveThreshold && emaFastBelowSlow

	return regularExitConditions
}

// executeEntry executa a operação de entrada
func (t *TrendMasterStrategy) executeEntry(df *core.Dataframe, broker core.Broker, quotePosition, closePrice float64) {
	pair := df.Pair

	// Cancelar qualquer ordem ativa anterior (garantia)
	if orderID, exists := t.activeOrders[pair]; exists {
		order, err := broker.Order(pair, orderID)
		if err == nil {
			_ = broker.Cancel(order)
		}
		delete(t.activeOrders, pair)
	}

	// Ajustar tamanho da posição com base no ATR para controle de risco
	atr := df.Metadata["atr"].Last(0)
	positionSize := t.positionSize

	// Reduzir tamanho da posição após perdas consecutivas
	if t.consecutiveLosses > 0 {
		// Reduzir tamanho da posição em 20% por cada perda consecutiva (até 60%)
		reductionFactor := 1.0 - float64(t.consecutiveLosses)*0.2
		if reductionFactor < 0.4 {
			reductionFactor = 0.4 // No mínimo 40% do tamanho original
		}
		positionSize *= reductionFactor
	}

	// Calcular stop loss baseado em ATR se disponível
	stopLossPrice := 0.0
	if atr > 0 {
		// Usar ATR para calcular o stop loss dinâmico
		stopLossPrice = closePrice - (atr * t.atrMultiplier)

		// Calcular percentual de stop loss baseado no ATR
		stopLossPercent := (closePrice - stopLossPrice) / closePrice

		// Se o stop loss baseado em ATR for maior que o máximo permitido, ajustar o tamanho da posição
		if stopLossPercent > t.stopLoss {
			// Ajustar o tamanho da posição para limitar o risco
			riskAdjustment := t.stopLoss / stopLossPercent
			positionSize *= riskAdjustment

			// Usar o stop loss fixo em vez do ATR
			stopLossPrice = closePrice * (1.0 - t.stopLoss)
		}
	} else {
		// Usar stop loss fixo se ATR não estiver disponível
		stopLossPrice = closePrice * (1.0 - t.stopLoss)
	}

	// Limitar risco por trade
	maxRiskAmount := quotePosition * t.maxRiskPerTrade
	actualRiskAmount := quotePosition * positionSize * ((closePrice - stopLossPrice) / closePrice)
	if actualRiskAmount > maxRiskAmount {
		// Ajustar o tamanho da posição para limitar o risco
		positionSize *= maxRiskAmount / actualRiskAmount
	}

	// Calcular tamanho da posição baseado no capital disponível
	entryAmount := quotePosition * positionSize

	// Executar ordem de compra a mercado
	_, err := broker.CreateOrderMarketQuote(core.SideTypeBuy, pair, entryAmount)
	if err != nil {
		backnrun.DefaultLog.WithFields(map[string]interface{}{
			"pair":  pair,
			"side":  core.SideTypeBuy,
			"quote": entryAmount,
			"price": closePrice,
		}).Error(err)
		return
	}

	// Registrar preço de entrada
	t.entryPrice[pair] = closePrice

	// Inicializar o trailing stop com o preço de entrada
	t.lastPrice[pair] = closePrice

	// Obter posição atualizada após a compra
	assetPosition, _, err := broker.Position(pair)
	if err != nil {
		backnrun.DefaultLog.Error(err)
		return
	}

	// Configurar take profit e stop loss com ordem OCO
	takeProfitPrice := closePrice * (1.0 + t.profitTarget)

	orders, err := broker.CreateOrderOCO(
		core.SideTypeSell,
		pair,
		assetPosition,
		takeProfitPrice,
		stopLossPrice,
		stopLossPrice,
	)

	if err != nil {
		backnrun.DefaultLog.WithFields(map[string]interface{}{
			"pair":       pair,
			"side":       core.SideTypeSell,
			"asset":      assetPosition,
			"takeProfit": takeProfitPrice,
			"stopPrice":  stopLossPrice,
		}).Error(err)
	} else if len(orders) > 0 {
		// Armazenar ID da primeira ordem para referência futura
		t.activeOrders[pair] = orders[0].ID
	}

	backnrun.DefaultLog.WithFields(map[string]interface{}{
		"pair":              pair,
		"entryPrice":        closePrice,
		"positionSize":      positionSize,
		"stopLossPrice":     stopLossPrice,
		"takeProfitPrice":   takeProfitPrice,
		"consecutiveLosses": t.consecutiveLosses,
	}).Info("Entrada executada")
}

// executeExit executa a operação de saída
func (t *TrendMasterStrategy) executeExit(df *core.Dataframe, broker core.Broker, assetPosition float64) {
	pair := df.Pair
	currentPrice := df.Close.Last(0)

	// Cancelar qualquer ordem OCO ativa
	if orderID, exists := t.activeOrders[pair]; exists {
		order, err := broker.Order(pair, orderID)
		if err == nil {
			err = broker.Cancel(order)
			if err != nil {
				backnrun.DefaultLog.WithFields(map[string]interface{}{
					"pair":    pair,
					"orderID": orderID,
				}).Error(err)
			}
		}
		// Remover a referência à ordem
		delete(t.activeOrders, pair)
	}

	// Vender toda a posição
	_, err := broker.CreateOrderMarket(core.SideTypeSell, pair, assetPosition)
	if err != nil {
		backnrun.DefaultLog.WithFields(map[string]interface{}{
			"pair":  pair,
			"side":  core.SideTypeSell,
			"asset": assetPosition,
			"price": currentPrice,
		}).Error(err)
		return
	}

	// Verificar resultado do trade
	entryPrice, exists := t.entryPrice[pair]
	if exists {
		tradeProfit := currentPrice > entryPrice
		t.lastTradeResult[pair] = tradeProfit

		// Atualizar contador de perdas consecutivas
		if tradeProfit {
			t.consecutiveLosses = 0 // Reset após um trade lucrativo
		} else {
			t.consecutiveLosses++ // Incrementar contador de perdas
		}

		backnrun.DefaultLog.WithFields(map[string]interface{}{
			"pair":        pair,
			"entryPrice":  entryPrice,
			"exitPrice":   currentPrice,
			"profit":      (currentPrice - entryPrice) / entryPrice * 100,
			"isProfit":    tradeProfit,
			"consecutive": t.consecutiveLosses,
		}).Info("Saída executada")

		// Limpar o preço de entrada
		delete(t.entryPrice, pair)
	}
}
