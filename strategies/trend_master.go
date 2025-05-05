package strategies

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/markcheno/go-talib"
	"github.com/raykavin/backnrun/bot"
	"github.com/raykavin/backnrun/core"
	"github.com/raykavin/backnrun/indicator"
)

// TrendMasterConfig contém todos os parâmetros configuráveis da estratégia
type TrendMasterConfig struct {
	// Timeframe base para a estratégia
	Timeframe string

	// Timeframe superior para filtro de tendência
	HigherTimeframe string

	// Período de aquecimento para cálculos históricos
	WarmupPeriod int

	// Máximo de operações por dia
	MaxTradesPerDay int

	// Controle de horário de trading
	TradingHoursEnabled bool
	TradingStartHour    int
	TradingEndHour      int

	// Parâmetros de indicadores EMA
	EmaFastPeriod int
	EmaSlowPeriod int
	EmaLongPeriod int

	// Parâmetros MACD
	MacdFastPeriod   int
	MacdSlowPeriod   int
	MacdSignalPeriod int

	// Parâmetros ADX
	AdxPeriod          int
	AdxThreshold       float64
	AdxMinimumDiSpread float64

	// Parâmetros RSI
	UseRsiFilter         bool
	RsiPeriod            int
	RsiOverbought        float64
	RsiOversold          float64
	RsiExtremeOverbought float64

	// Parâmetros ATR
	AtrPeriod           int
	AtrMultiplier       float64
	VolatilityThreshold float64

	// Parâmetros Volume
	UseVolFilter bool
	VolAvgPeriod int
	VolMinRatio  float64

	// Controle de entrada
	UseHigherTfConfirmation bool

	// Filtro de sentimento
	UseSentimentFilter bool
	SentimentThreshold float64

	// Correlação de mercado
	UseMarketCorrelation         bool
	CorrelationReferenceSymbol   string
	CorrelationPeriod            int
	NegativeCorrelationThreshold float64

	// Gestão de posição
	PositionSize        float64
	MaxRiskPerTrade     float64
	TrailingStopPercent float64

	// Tamanho adaptativo de posição
	UseAdaptiveSize       bool
	WinIncreaseFactor     float64
	LossReductionFactor   float64
	MinPositionSizeFactor float64
	MaxPositionSizeFactor float64

	// Saída parcial
	UsePartialTakeProfit bool
	PartialExitLevels    []PartialExitLevel

	// Alvos dinâmicos
	UseDynamicTargets bool
	BaseTarget        float64
	AtrTargetFactor   float64
	MinTarget         float64
	MaxTarget         float64

	// Saídas rápidas
	UseMacdReversalExit   bool
	MacdReversalThreshold float64
	UseAdxFallingExit     bool
	UsePriceActionExit    bool

	// Parâmetros específicos por tipo de mercado
	MarketSpecificSettings map[string]MarketSpecificConfig
}

// PartialExitLevel define um nível de saída parcial
type PartialExitLevel struct {
	Percentage   float64
	Target       float64
	TrailingOnly bool
}

// MarketSpecificConfig contém configurações específicas para cada tipo de mercado
type MarketSpecificConfig struct {
	VolatilityThreshold float64
	TrailingStopPercent float64
	AtrMultiplier       float64
}

// PartialPosition armazena detalhes de uma posição parcial
type PartialPosition struct {
	Quantity   float64
	EntryPrice float64
	OrderID    int64
	Level      int
}

// TrendMaster implementa uma estratégia que combina múltiplos indicadores
// para identificar tendências fortes e gerar sinais de entrada e saída
type TrendMaster struct {
	// Configuração da estratégia
	config TrendMasterConfig

	// Estado interno da estratégia
	marketType             map[string]string // "crypto", "forex", "stocks"
	higherTfCache          map[string]*core.Dataframe
	higherTfLastUpdate     map[string]time.Time
	marketSentiment        map[string]float64
	correlationValues      map[string][]float64
	correlationRef         map[string][]float64
	marketCorrelation      map[string]float64
	winCount               int
	lossCount              int
	winStreak              int
	lossStreak             int
	consecutiveLosses      int
	dailyTradeCount        map[string]int
	lastTradeDate          string
	lastTradeResult        map[string]bool
	partialPositions       map[string][]PartialPosition
	partialOrders          map[string][]int64
	activeOrders           map[string]map[int]int64
	lastPrice              map[string]float64
	entryPrice             map[string]float64
	positionSize           map[string]float64
	isDataFrameInitialized map[string]bool
}

// NewTrendMasterStrategy cria uma nova instância da estratégia com parâmetros definidos
func NewTrendMaster(config TrendMasterConfig) *TrendMaster {
	return &TrendMaster{
		config:                 config,
		marketType:             make(map[string]string),
		higherTfCache:          make(map[string]*core.Dataframe),
		higherTfLastUpdate:     make(map[string]time.Time),
		marketSentiment:        make(map[string]float64),
		correlationValues:      make(map[string][]float64),
		correlationRef:         make(map[string][]float64),
		marketCorrelation:      make(map[string]float64),
		dailyTradeCount:        make(map[string]int),
		lastTradeResult:        make(map[string]bool),
		partialPositions:       make(map[string][]PartialPosition),
		partialOrders:          make(map[string][]int64),
		activeOrders:           make(map[string]map[int]int64),
		lastPrice:              make(map[string]float64),
		entryPrice:             make(map[string]float64),
		positionSize:           make(map[string]float64),
		isDataFrameInitialized: make(map[string]bool),
	}
}

// Timeframe retorna o timeframe requerido para esta estratégia
func (t TrendMaster) Timeframe() string {
	return t.config.Timeframe
}

// WarmupPeriod retorna o número de candles necessários antes da estratégia estar pronta
func (t TrendMaster) WarmupPeriod() int {
	return t.config.WarmupPeriod
}

// Indicators calcula e retorna os indicadores usados por esta estratégia
func (t TrendMaster) Indicators(df *core.Dataframe) []core.ChartIndicator {
	// Verificar se já temos dados para este par
	if !t.isDataFrameInitialized[df.Pair] {
		t.isDataFrameInitialized[df.Pair] = true

		// Inicializar arrays para correlação
		t.correlationValues[df.Pair] = make([]float64, 0, t.config.CorrelationPeriod*2)

		// Se estamos usando correlação e este é o par de referência
		if t.config.UseMarketCorrelation && strings.Contains(df.Pair, t.config.CorrelationReferenceSymbol) {
			t.correlationRef[df.Pair] = make([]float64, 0, t.config.CorrelationPeriod*2)
		}
	}

	// Calcular EMAs
	df.Metadata["ema_fast"] = indicator.EMA(df.Close, t.config.EmaFastPeriod)
	df.Metadata["ema_slow"] = indicator.EMA(df.Close, t.config.EmaSlowPeriod)
	df.Metadata["ema_long_high"] = indicator.EMA(df.High, t.config.EmaLongPeriod)
	df.Metadata["ema_long_low"] = indicator.EMA(df.Low, t.config.EmaLongPeriod)

	// Calcular MACD
	df.Metadata["macd"], df.Metadata["macd_signal"], df.Metadata["macd_hist"] = indicator.MACD(
		df.Close,
		t.config.MacdFastPeriod,
		t.config.MacdSlowPeriod,
		t.config.MacdSignalPeriod,
	)

	// Calcular ADX e indicadores direcionais
	df.Metadata["adx"] = indicator.ADX(df.High, df.Low, df.Close, t.config.AdxPeriod)
	df.Metadata["plus_di"] = talib.PlusDI(df.High, df.Low, df.Close, t.config.AdxPeriod)
	df.Metadata["minus_di"] = talib.MinusDI(df.High, df.Low, df.Close, t.config.AdxPeriod)

	// Adicionar ATR para cálculo de volatilidade
	df.Metadata["atr"] = talib.Atr(df.High, df.Low, df.Close, t.config.AtrPeriod)

	// Calcular RSI para filtragem adicional
	if t.config.UseRsiFilter {
		df.Metadata["rsi"] = talib.Rsi(df.Close, t.config.RsiPeriod)
	}

	// Calcular média de volume para filtragem
	if t.config.UseVolFilter {
		df.Metadata["vol_avg"] = indicator.SMA(df.Volume, t.config.VolAvgPeriod)
	}

	// Atualizar cache de dados para timeframe superior
	t.updateHigherTimeframeCache(df)

	// Atualizar dados para correlação
	t.updateCorrelationData(df)

	// Detectar tipo de mercado (se ainda não foi detectado)
	t.detectMarketType(df.Pair)

	// Retornar indicadores para visualização
	return []core.ChartIndicator{
		{
			Overlay:   true,
			GroupName: "Moving Averages",
			Time:      df.Time,
			Metrics: []core.IndicatorMetric{
				{
					Values: df.Metadata["ema_fast"],
					Name:   "EMA " + string(rune(t.config.EmaFastPeriod)),
					Color:  "blue",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["ema_slow"],
					Name:   "EMA " + string(rune(t.config.EmaSlowPeriod)),
					Color:  "green",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["ema_long_high"],
					Name:   "EMA " + string(rune(t.config.EmaLongPeriod)) + " (High)",
					Color:  "purple",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["ema_long_low"],
					Name:   "EMA " + string(rune(t.config.EmaLongPeriod)) + " (Low)",
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

// updateHigherTimeframeCache atualiza o cache de dataframe do timeframe superior
func (t *TrendMaster) updateHigherTimeframeCache(df *core.Dataframe) {
	// Se não estamos usando filtro de timeframe superior, não precisamos atualizar o cache
	if !t.config.UseHigherTfConfirmation {
		return
	}

	// Verificar se é hora de atualizar o cache (a cada 5 minutos)
	now := time.Now()
	lastUpdate, exists := t.higherTfLastUpdate[df.Pair]
	if exists && now.Sub(lastUpdate) < time.Minute*5 {
		return
	}

	// Criar novo dataframe para timeframe superior se não existir
	if _, exists := t.higherTfCache[df.Pair]; !exists {
		t.higherTfCache[df.Pair] = &core.Dataframe{
			Pair:     df.Pair,
			Metadata: make(map[string]core.Series[float64]),
		}
	}

	// Usar as listas atuais do df para criar um dataframe de timeframe superior
	t.processHigherTimeframeData(df)

	// Atualizar timestamp da última atualização
	t.higherTfLastUpdate[df.Pair] = now
}

// processHigherTimeframeData processa dados para o dataframe de timeframe superior
func (t *TrendMaster) processHigherTimeframeData(df *core.Dataframe) {
	// Função para criar candles de timeframe superior a partir de candles menores
	higherDf := t.higherTfCache[df.Pair]

	// Configuração de períodos para o timeframe superior
	timeframeMap := map[string]int{
		"1m":  1,
		"3m":  3,
		"5m":  5,
		"15m": 15,
		"30m": 30,
		"1h":  60,
		"2h":  120,
		"4h":  240,
		"6h":  360,
		"8h":  480,
		"12h": 720,
		"1d":  1440,
	}

	// Obter número de minutos para cada timeframe
	baseMinutes, existsBase := timeframeMap[t.config.Timeframe]
	higherMinutes, existsHigher := timeframeMap[t.config.HigherTimeframe]

	if !existsBase || !existsHigher {
		return // Timeframes inválidos
	}

	// Número de candles do timeframe base que formam um candle do timeframe superior
	candlesPerHigherTf := higherMinutes / baseMinutes

	// Garantir que temos dados suficientes
	if len(df.Close) < candlesPerHigherTf {
		return
	}

	// Agrupar candles do timeframe base em candles do timeframe superior
	numCandles := len(df.Close) / candlesPerHigherTf

	// Inicializar slices para o dataframe de timeframe superior
	higherDf.Time = make([]time.Time, numCandles)
	higherDf.Open = make(core.Series[float64], numCandles)
	higherDf.High = make(core.Series[float64], numCandles)
	higherDf.Low = make(core.Series[float64], numCandles)
	higherDf.Close = make(core.Series[float64], numCandles)
	higherDf.Volume = make(core.Series[float64], numCandles)

	// Preencher o dataframe de timeframe superior
	for i := 0; i < numCandles; i++ {
		startIdx := i * candlesPerHigherTf
		endIdx := (i + 1) * candlesPerHigherTf
		if endIdx > len(df.Close) {
			endIdx = len(df.Close)
		}

		// Período atual de candles
		periodOpen := df.Open[startIdx]
		periodClose := df.Close[endIdx-1]
		periodVolume := 0.0

		// Encontrar máximos e mínimos no período
		periodHigh := df.High[startIdx]
		periodLow := df.Low[startIdx]

		for j := startIdx; j < endIdx; j++ {
			if df.High[j] > periodHigh {
				periodHigh = df.High[j]
			}
			if df.Low[j] < periodLow {
				periodLow = df.Low[j]
			}
			periodVolume += df.Volume[j]
		}

		// Preencher o dataframe de timeframe superior
		higherDf.Time[i] = df.Time[startIdx]
		higherDf.Open[i] = periodOpen
		higherDf.High[i] = periodHigh
		higherDf.Low[i] = periodLow
		higherDf.Close[i] = periodClose
		higherDf.Volume[i] = periodVolume
	}

	// Calcular indicadores para o timeframe superior
	higherDf.Metadata["ema_fast"] = indicator.EMA(higherDf.Close, t.config.EmaFastPeriod)
	higherDf.Metadata["ema_slow"] = indicator.EMA(higherDf.Close, t.config.EmaSlowPeriod)
	higherDf.Metadata["ema_long"] = indicator.EMA(higherDf.Close, t.config.EmaLongPeriod)
	higherDf.Metadata["macd"], higherDf.Metadata["macd_signal"], higherDf.Metadata["macd_hist"] = indicator.MACD(
		higherDf.Close,
		t.config.MacdFastPeriod,
		t.config.MacdSlowPeriod,
		t.config.MacdSignalPeriod,
	)
	higherDf.Metadata["adx"] = indicator.ADX(higherDf.High, higherDf.Low, higherDf.Close, t.config.AdxPeriod)
	higherDf.Metadata["plus_di"] = talib.PlusDI(higherDf.High, higherDf.Low, higherDf.Close, t.config.AdxPeriod)
	higherDf.Metadata["minus_di"] = talib.MinusDI(higherDf.High, higherDf.Low, higherDf.Close, t.config.AdxPeriod)
}

// updateCorrelationData atualiza os dados de correlação com o mercado global
func (t *TrendMaster) updateCorrelationData(df *core.Dataframe) {
	// Se não estamos usando correlação, retornar
	if !t.config.UseMarketCorrelation {
		return
	}

	// Verificar se temos close price válido
	if len(df.Close) == 0 {
		return
	}

	// Adicionar último close price ao array de valores
	lastClose := df.Close.Last(0)
	if lastClose > 0 {
		t.correlationValues[df.Pair] = append(t.correlationValues[df.Pair], lastClose)

		// Limitar o tamanho do array
		if len(t.correlationValues[df.Pair]) > t.config.CorrelationPeriod*2 {
			t.correlationValues[df.Pair] = t.correlationValues[df.Pair][1:]
		}
	}

	// Se este par contém o símbolo de referência, atualizar os dados de referência
	if strings.Contains(df.Pair, t.config.CorrelationReferenceSymbol) {
		if lastClose > 0 {
			t.correlationRef[df.Pair] = append(t.correlationRef[df.Pair], lastClose)

			// Limitar o tamanho do array
			if len(t.correlationRef[df.Pair]) > t.config.CorrelationPeriod*2 {
				t.correlationRef[df.Pair] = t.correlationRef[df.Pair][1:]
			}
		}

		// Atualizar a correlação para todos os pares
		t.calculateAllCorrelations()
	}
}

// calculateAllCorrelations calcula a correlação entre todos os pares e o par de referência
func (t *TrendMaster) calculateAllCorrelations() {
	// Encontrar o par de referência
	var refPair string
	var refValues []float64

	for pair, values := range t.correlationRef {
		if strings.Contains(pair, t.config.CorrelationReferenceSymbol) && len(values) >= t.config.CorrelationPeriod {
			refPair = pair
			refValues = values
			break
		}
	}

	// Se não encontramos o par de referência com dados suficientes, retornar
	if refPair == "" || len(refValues) < t.config.CorrelationPeriod {
		return
	}

	// Usar apenas os últimos N valores
	refValues = refValues[len(refValues)-t.config.CorrelationPeriod:]

	// Calcular correlação para cada par
	for pair, values := range t.correlationValues {
		// Se não temos dados suficientes, pular
		if len(values) < t.config.CorrelationPeriod {
			continue
		}

		// Usar apenas os últimos N valores
		pairValues := values[len(values)-t.config.CorrelationPeriod:]

		// Calcular correlação
		correlation := t.calculateCorrelation(refValues, pairValues)

		// Armazenar correlação
		t.marketCorrelation[pair] = correlation
	}
}

// calculateCorrelation calcula a correlação entre dois arrays de valores
func (t *TrendMaster) calculateCorrelation(x, y []float64) float64 {
	// Verificar se os arrays têm o mesmo tamanho
	if len(x) != len(y) {
		return 0
	}

	// Calcular médias
	var sumX, sumY, sumXY, sumX2, sumY2 float64
	n := float64(len(x))

	for i := 0; i < len(x); i++ {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
		sumY2 += y[i] * y[i]
	}

	// Calcular correlação de Pearson
	numerator := sumXY - sumX*sumY/n
	denominator := math.Sqrt((sumX2 - sumX*sumX/n) * (sumY2 - sumY*sumY/n))

	if denominator == 0 {
		return 0
	}

	return numerator / denominator
}

// OnCandle é chamado para cada novo candle e implementa a lógica de trading
func (t *TrendMaster) OnCandle(ctx context.Context, df *core.Dataframe, broker core.Broker) {
	pair := df.Pair
	closePrice := df.Close.Last(0)

	// Verificar se o dataframe está inicializado
	if !t.isDataFrameInitialized[pair] {
		t.isDataFrameInitialized[pair] = true

		// Inicializar arrays para correlação
		t.correlationValues[pair] = make([]float64, 0, t.config.CorrelationPeriod*2)

		// Se estamos usando correlação e este é o par de referência
		if t.config.UseMarketCorrelation && strings.Contains(pair, t.config.CorrelationReferenceSymbol) {
			t.correlationRef[pair] = make([]float64, 0, t.config.CorrelationPeriod*2)
		}
	}

	// Obter data atual (último candle)
	currentDate := ""
	if len(df.Time) > 0 {
		currentDate = df.Time[len(df.Time)-1].Format("2006-01-02")
	}

	// Reiniciar contagem diária de operações se estivermos em um novo dia
	if currentDate != t.lastTradeDate {
		t.dailyTradeCount = make(map[string]int)
		t.lastTradeDate = currentDate
	}

	// Verificar se está dentro do horário de trading permitido
	if t.config.TradingHoursEnabled {
		currentHour := time.Now().UTC().Hour()
		if currentHour < t.config.TradingStartHour || currentHour >= t.config.TradingEndHour {
			// Fora do horário de trading, não realizar novas operações
			return
		}
	}

	// Obter posição atual
	assetPosition, quotePosition, err := broker.Position(ctx, pair)
	if err != nil {
		bot.DefaultLog.Error(err)
		return
	}

	// Detectar tipo de mercado
	t.detectMarketType(pair)

	// Atualizar trailing stop se em posição
	if assetPosition > 0 {
		lastPrice, exists := t.lastPrice[pair]
		if !exists || closePrice > lastPrice {
			t.lastPrice[pair] = closePrice
		}
	}

	// Verificar sinais de entrada e saída
	if assetPosition > 0 {
		// Já estamos em posição longa, verificar saída parcial
		if t.config.UsePartialTakeProfit {
			t.checkPartialExits(ctx, df, broker, assetPosition, pair)
		}

		// Verificar saída completa
		if t.shouldExit(df) || t.checkTrailingStop(df, pair) {
			t.executeExit(ctx, df, broker, assetPosition)
			// Reiniciar após saída
			delete(t.lastPrice, pair)
		}
	} else {
		// Sem posição, verificar entrada (apenas se não excedemos o limite diário)
		tradeCount := t.dailyTradeCount[pair]
		if tradeCount < t.config.MaxTradesPerDay && t.shouldEnter(df) {
			t.executeEntry(ctx, df, broker, quotePosition, closePrice)
			// Incrementar contador de operações
			t.dailyTradeCount[pair] = tradeCount + 1
		}
	}

	// Atualizar cache de data (para a próxima verificação de dia)
	t.lastTradeDate = currentDate
}

// detectMarketType detecta o tipo de mercado com base no par
func (t *TrendMaster) detectMarketType(pair string) {
	// Se já temos o tipo de mercado armazenado, usar o valor armazenado
	if _, exists := t.marketType[pair]; exists {
		return
	}

	// Detectar tipo de mercado com base no padrão do par
	if strings.HasSuffix(pair, "USDT") || strings.HasSuffix(pair, "BTC") || strings.HasSuffix(pair, "ETH") ||
		strings.HasSuffix(pair, "BNB") || strings.Contains(pair, "PERP") {
		t.marketType[pair] = "crypto"
	} else if strings.HasSuffix(pair, "USD") || strings.Contains(pair, "JPY") ||
		strings.Contains(pair, "EUR") || strings.Contains(pair, "GBP") ||
		strings.Contains(pair, "AUD") || strings.Contains(pair, "CAD") ||
		strings.Contains(pair, "CHF") || strings.Contains(pair, "NZD") {
		t.marketType[pair] = "forex"
	} else {
		t.marketType[pair] = "stocks"
	}
}

// getMarketSpecificConfig retorna a configuração específica para o tipo de mercado
func (t *TrendMaster) getMarketSpecificConfig(pair string) MarketSpecificConfig {
	marketType, exists := t.marketType[pair]
	if !exists {
		// Tipo de mercado padrão
		marketType = "crypto"
	}

	// Verificar se temos configuração específica para este tipo de mercado
	if config, exists := t.config.MarketSpecificSettings[marketType]; exists {
		return config
	}

	// Retornar configuração padrão
	return MarketSpecificConfig{
		VolatilityThreshold: t.config.VolatilityThreshold,
		TrailingStopPercent: t.config.TrailingStopPercent,
		AtrMultiplier:       t.config.AtrMultiplier,
	}
}

// checkTrailingStop verifica se o trailing stop foi atingido
func (t *TrendMaster) checkTrailingStop(df *core.Dataframe, pair string) bool {
	closePrice := df.Close.Last(0)
	lastPrice, exists := t.lastPrice[pair]

	if !exists {
		return false
	}

	// Obter configuração específica do mercado
	marketConfig := t.getMarketSpecificConfig(pair)
	trailStopPercent := marketConfig.TrailingStopPercent

	// Se o preço caiu abaixo do trailing stop, acionar saída
	trailAmount := lastPrice * trailStopPercent
	if closePrice <= lastPrice-trailAmount {
		bot.DefaultLog.WithFields(map[string]any{
			"pair":         pair,
			"highestPrice": lastPrice,
			"currentPrice": closePrice,
			"trailAmount":  trailAmount,
		}).Info("Trailing stop activated")
		return true
	}

	return false
}

// checkHigherTimeframeTrend verifica a tendência no timeframe superior
func (t *TrendMaster) checkHigherTimeframeTrend(pair string) bool {
	// Se não estamos usando filtro de timeframe superior, retornar true
	if !t.config.UseHigherTfConfirmation {
		return true
	}

	// Verificar se temos cache do timeframe superior
	higherDf, exists := t.higherTfCache[pair]
	if !exists || len(higherDf.Close) == 0 {
		return true // Sem dados suficientes, permitir entrada
	}

	// Verificar condições no timeframe superior
	emaFast := higherDf.Metadata["ema_fast"].Last(0)
	emaSlow := higherDf.Metadata["ema_slow"].Last(0)
	emaLong := higherDf.Metadata["ema_long"].Last(0)
	macd := higherDf.Metadata["macd"].Last(0)
	macdSignal := higherDf.Metadata["macd_signal"].Last(0)
	plusDI := higherDf.Metadata["plus_di"].Last(0)
	minusDI := higherDf.Metadata["minus_di"].Last(0)
	adx := higherDf.Metadata["adx"].Last(0)

	// Verificar tendência de alta no timeframe superior
	emaAlignment := emaFast > emaSlow && emaSlow > emaLong
	macdBullish := macd > macdSignal
	diPositive := plusDI > minusDI
	strongTrend := adx > t.config.AdxThreshold

	// Pelo menos 3 das 4 condições devem ser verdadeiras
	conditionsCount := 0
	if emaAlignment {
		conditionsCount++
	}
	if macdBullish {
		conditionsCount++
	}
	if diPositive {
		conditionsCount++
	}
	if strongTrend {
		conditionsCount++
	}

	return conditionsCount >= 3
}

// shouldEnter verifica se as condições de entrada são atendidas
func (t *TrendMaster) shouldEnter(df *core.Dataframe) bool {
	pair := df.Pair
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
	adxAboveThreshold := adx > t.config.AdxThreshold
	emaFastAboveSlow := emaFast > emaSlow
	diSpreadSufficient := (plusDI - minusDI) > t.config.AdxMinimumDiSpread

	// Obter configuração específica do mercado
	marketConfig := t.getMarketSpecificConfig(pair)

	// Filtro de volatilidade: verificar se a volatilidade não é muito alta
	volatilityCheck := true
	if atr > 0 {
		volatilityRatio := atr / closePrice
		volatilityCheck = volatilityRatio < marketConfig.VolatilityThreshold
	}

	// Filtro RSI: evitar compra em mercado sobrecomprado
	rsiCheck := true
	if t.config.UseRsiFilter {
		rsi := df.Metadata["rsi"].Last(0)
		rsiCheck = rsi < t.config.RsiOverbought // RSI não está sobrecomprado
	}

	// Filtro de volume: verificar se o volume é suficiente
	volumeCheck := true
	if t.config.UseVolFilter {
		volume := df.Volume.Last(0)
		volumeAvg := df.Metadata["vol_avg"].Last(0)
		if volumeAvg > 0 {
			volumeRatio := volume / volumeAvg
			volumeCheck = volumeRatio >= t.config.VolMinRatio // Volume acima da média
		}
	}

	// Verificar confirmação de timeframe superior
	higherTfCheck := t.checkHigherTimeframeTrend(pair)

	// Verificar filtro de sentimento de mercado
	sentimentCheck := true
	if t.config.UseSentimentFilter {
		sentiment, exists := t.marketSentiment[pair]
		if exists {
			// Valores mais baixos são mais restritivos (mais medo no mercado)
			if sentiment < t.config.SentimentThreshold {
				// Em mercados com medo alto, exigir condições mais fortes
				sentimentCheck = adx > (t.config.AdxThreshold+5) && (plusDI-minusDI) > 15
			}
		}
	}

	// Verificar correlação de mercado
	correlationCheck := true
	if t.config.UseMarketCorrelation {
		correlation, exists := t.marketCorrelation[pair]
		if exists {
			// Se correlação fortemente negativa com o mercado, ser mais cauteloso
			if correlation < t.config.NegativeCorrelationThreshold {
				correlationCheck = false // Evitar operar contra o mercado global
			}
		}
	}

	// Verificar tendência consistentemente de alta
	bullishMarket := true

	// Reduzir frequência de operação após perdas consecutivas
	riskAdjustment := true
	if t.consecutiveLosses >= 2 {
		// Após 2 perdas consecutivas, exigir condições mais rígidas
		riskAdjustment = adx > (t.config.AdxThreshold+5) && (plusDI-minusDI) > 10
	}

	// Todas as condições principais devem ser verdadeiras para entrada
	mainConditions := priceAboveEMA && macdAboveSignal && plusDIAboveMinusDI &&
		adxAboveThreshold && emaFastAboveSlow && diSpreadSufficient

	// Filtros adicionais para melhorar qualidade do sinal
	additionalFilters := volatilityCheck && rsiCheck && volumeCheck &&
		bullishMarket && riskAdjustment && higherTfCheck &&
		sentimentCheck && correlationCheck

	return mainConditions && additionalFilters
}

// shouldExit verifica se as condições de saída são atendidas
func (t *TrendMaster) shouldExit(df *core.Dataframe) bool {
	closePrice := df.Close.Last(0)
	emaLongLow := df.Metadata["ema_long_low"].Last(0)
	emaFast := df.Metadata["ema_fast"].Last(0)
	emaSlow := df.Metadata["ema_slow"].Last(0)
	macd := df.Metadata["macd"].Last(0)
	macdSignal := df.Metadata["macd_signal"].Last(0)
	plusDI := df.Metadata["plus_di"].Last(0)
	minusDI := df.Metadata["minus_di"].Last(0)
	adx := df.Metadata["adx"].Last(0)

	// Condições básicas de saída
	priceBelowEMA := closePrice < emaLongLow
	macdBelowSignal := macd < macdSignal
	minusDIAbovePlusDI := minusDI > plusDI
	adxAboveThreshold := adx > t.config.AdxThreshold
	emaFastBelowSlow := emaFast < emaSlow

	// Verificar quebra da média móvel longa calculada sobre os mínimos
	// Este é um sinal de saída rápida independente de outras condições
	if priceBelowEMA && t.config.UsePriceActionExit {
		bot.DefaultLog.WithFields(map[string]any{
			"pair":       df.Pair,
			"closePrice": closePrice,
			"emaLongLow": emaLongLow,
		}).Info("Quick exit: price below EMA Low")
		return true
	}

	// Saída rápida se MACD cair muito rapidamente (potencial reversão forte)
	if t.config.UseMacdReversalExit {
		macdHist := df.Metadata["macd_hist"].Last(0)
		prevMacdHist := df.Metadata["macd_hist"].Last(1)
		if macdHist < 0 && prevMacdHist > 0 && macdHist < prevMacdHist*-t.config.MacdReversalThreshold {
			bot.DefaultLog.WithFields(map[string]any{
				"pair":         df.Pair,
				"macdHist":     macdHist,
				"prevMacdHist": prevMacdHist,
			}).Info("Quick exit: MACD strongly reversing")
			return true
		}
	}

	// Saída rápida se ADX começar a cair enquanto em posição (tendência enfraquecendo)
	if t.config.UseAdxFallingExit {
		adxFalling := adx < df.Metadata["adx"].Last(1) && df.Metadata["adx"].Last(1) < df.Metadata["adx"].Last(2)
		if adxFalling && minusDIAbovePlusDI {
			bot.DefaultLog.WithFields(map[string]any{
				"pair":    df.Pair,
				"adx":     adx,
				"prevAdx": df.Metadata["adx"].Last(1),
				"plusDI":  plusDI,
				"minusDI": minusDI,
			}).Info("Quick exit: ADX falling and -DI above +DI")
			return true
		}
	}

	// RSI indicando sobrecompra extrema
	if t.config.UseRsiFilter {
		rsi := df.Metadata["rsi"].Last(0)
		if rsi > t.config.RsiExtremeOverbought { // Sobrecompra extrema
			bot.DefaultLog.WithFields(map[string]any{
				"pair": df.Pair,
				"rsi":  rsi,
			}).Info("Quick exit: RSI in extreme overbought")
			return true
		}
	}

	// Todas as condições regulares de saída
	regularExitConditions := macdBelowSignal && minusDIAbovePlusDI && adxAboveThreshold && emaFastBelowSlow

	return regularExitConditions
}

// checkPartialExits verifica e executa saídas parciais
func (t *TrendMaster) checkPartialExits(ctx context.Context, df *core.Dataframe, broker core.Broker, assetPosition float64, pair string) {
	// Se não usamos saída parcial ou não temos níveis configurados, retornar
	if !t.config.UsePartialTakeProfit || len(t.config.PartialExitLevels) == 0 {
		return
	}

	closePrice := df.Close.Last(0)
	entryPrice, exists := t.entryPrice[pair]

	// Se não temos preço de entrada, não podemos verificar saídas parciais
	if !exists {
		return
	}

	// Verificar se já temos posições parciais para este par
	positions, posExists := t.partialPositions[pair]
	if !posExists {
		// Criar array para rastrear posições parciais
		t.partialPositions[pair] = make([]PartialPosition, 0)

		// Para cada nível configurado, criar uma posição parcial
		totalPosition := assetPosition
		for i, level := range t.config.PartialExitLevels {
			// Calcular quantidade para este nível
			levelQuantity := assetPosition * level.Percentage

			// Criar posição parcial
			partialPos := PartialPosition{
				Quantity:   levelQuantity,
				EntryPrice: entryPrice,
				Level:      i,
			}

			// Adicionar à lista de posições parciais
			t.partialPositions[pair] = append(t.partialPositions[pair], partialPos)

			// Verificar se já atingimos o alvo para este nível
			if !level.TrailingOnly {
				targetPrice := entryPrice * (1.0 + level.Target)

				// Se o preço atual já atingiu o alvo, executar saída parcial
				if closePrice >= targetPrice {
					t.executePartialExit(ctx, df, broker, pair, i, partialPos.Quantity, "Target reached")
				}
			}

			totalPosition -= levelQuantity
		}

		positions = t.partialPositions[pair]
	}

	// Para cada posição parcial existente
	for i, pos := range positions {
		// Pular posições já executadas
		if pos.Quantity <= 0 {
			continue
		}

		// Obter nível configurado
		level := t.config.PartialExitLevels[pos.Level]

		// Se não é trailing only, verificar alvo de preço
		if !level.TrailingOnly {
			targetPrice := pos.EntryPrice * (1.0 + level.Target)

			// Se o preço atual atingiu o alvo, executar saída parcial
			if closePrice >= targetPrice {
				t.executePartialExit(ctx, df, broker, pair, i, pos.Quantity, "Target reached")
			}
		} else {
			// Para níveis apenas com trailing stop, verificar trailing stop
			// Usamos o último preço mais alto registrado
			lastHighestPrice, exists := t.lastPrice[pair]
			if exists {
				// Calcular preço de trailing stop
				trailAmount := lastHighestPrice * t.config.TrailingStopPercent
				trailingStopPrice := lastHighestPrice - trailAmount

				// Se o preço atual caiu abaixo do trailing stop, executar saída parcial
				if closePrice <= trailingStopPrice {
					t.executePartialExit(ctx, df, broker, pair, i, pos.Quantity, "Trailing stop hit")
				}
			}
		}
	}
}

// executePartialExit executa uma saída parcial
func (t *TrendMaster) executePartialExit(ctx context.Context, df *core.Dataframe, broker core.Broker, pair string, posIndex int, quantity float64, reason string) {
	// Não executar se quantidade é zero ou negativa
	if quantity <= 0 {
		return
	}

	// Executar ordem de venda a mercado para a quantidade parcial
	order, err := broker.CreateOrderMarket(ctx, core.SideTypeSell, pair, quantity)
	if err != nil {
		bot.DefaultLog.WithFields(map[string]any{
			"pair":     pair,
			"quantity": quantity,
			"level":    posIndex,
			"error":    err,
		}).Error("Failed to execute partial exit")
		return
	}

	// Atualizar posição parcial
	t.partialPositions[pair][posIndex].Quantity = 0
	t.partialPositions[pair][posIndex].OrderID = order.ID

	// Registrar saída parcial
	bot.DefaultLog.WithFields(map[string]any{
		"pair":     pair,
		"quantity": quantity,
		"level":    posIndex,
		"orderID":  order.ID,
		"reason":   reason,
		"price":    df.Close.Last(0),
	}).Info("Partial exit executed")
}

// executeEntry executa a operação de entrada
func (t *TrendMaster) executeEntry(ctx context.Context, df *core.Dataframe, broker core.Broker, quotePosition, closePrice float64) {
	pair := df.Pair

	// Cancelar qualquer ordem ativa anterior
	if ordersMap, exists := t.activeOrders[pair]; exists {
		for _, orderID := range ordersMap {
			order, err := broker.Order(ctx, pair, orderID)
			if err == nil {
				_ = broker.Cancel(ctx, order)
			}
		}
		delete(t.activeOrders, pair)
	}

	// Obter configuração específica do mercado
	marketConfig := t.getMarketSpecificConfig(pair)

	// Ajustar tamanho da posição com base no ATR para controle de risco
	atr := df.Metadata["atr"].Last(0)
	positionSize := t.config.PositionSize

	// Ajustar tamanho da posição com base no desempenho recente (se habilitado)
	if t.config.UseAdaptiveSize {
		// Aumentar o tamanho após vitórias consecutivas
		if t.winStreak > 0 {
			increaseFactor := math.Min(float64(t.winStreak)*t.config.WinIncreaseFactor,
				t.config.MaxPositionSizeFactor-1.0)
			positionSize *= (1.0 + increaseFactor)

			// Limitar ao tamanho máximo
			if positionSize > t.config.PositionSize*t.config.MaxPositionSizeFactor {
				positionSize = t.config.PositionSize * t.config.MaxPositionSizeFactor
			}
		}

		// Reduzir o tamanho após perdas consecutivas
		if t.consecutiveLosses > 0 {
			// Reduzir tamanho para cada perda consecutiva
			reductionFactor := 1.0 - float64(t.consecutiveLosses)*t.config.LossReductionFactor
			if reductionFactor < t.config.MinPositionSizeFactor {
				reductionFactor = t.config.MinPositionSizeFactor
			}
			positionSize *= reductionFactor
		}
	}

	// Calcular stop loss baseado em ATR se disponível
	stopLossPrice := 0.0
	if atr > 0 {
		// Usar ATR para calcular stop loss dinâmico
		stopLossPrice = closePrice - (atr * marketConfig.AtrMultiplier)

		// Calcular percentual de stop loss baseado em ATR
		stopLossPercent := (closePrice - stopLossPrice) / closePrice

		// Se o stop loss baseado em ATR for maior que o máximo permitido, ajustar tamanho da posição
		if stopLossPercent > t.config.MaxRiskPerTrade {
			// Ajustar tamanho da posição para limitar risco
			riskAdjustment := t.config.MaxRiskPerTrade / stopLossPercent
			positionSize *= riskAdjustment
		}
	} else {
		// Usar stop loss fixo se ATR não estiver disponível
		stopLossPrice = closePrice * (1.0 - t.config.MaxRiskPerTrade)
	}

	// Limitar risco por operação
	maxRiskAmount := quotePosition * t.config.MaxRiskPerTrade
	actualRiskAmount := quotePosition * positionSize * ((closePrice - stopLossPrice) / closePrice)
	if actualRiskAmount > maxRiskAmount {
		// Ajustar tamanho da posição para limitar risco
		positionSize *= maxRiskAmount / actualRiskAmount
	}

	// Calcular tamanho da posição com base no capital disponível
	entryAmount := quotePosition * positionSize

	// Executar ordem de compra a mercado
	_, err := broker.CreateOrderMarketQuote(ctx, core.SideTypeBuy, pair, entryAmount)
	if err != nil {
		bot.DefaultLog.WithFields(map[string]any{
			"pair":  pair,
			"side":  core.SideTypeBuy,
			"quote": entryAmount,
			"price": closePrice,
		}).Error(err)
		return
	}

	// Registrar preço de entrada
	t.entryPrice[pair] = closePrice

	// Inicializar trailing stop com preço de entrada
	t.lastPrice[pair] = closePrice

	// Armazenar tamanho da posição para uso em saídas parciais
	t.positionSize[pair] = positionSize

	// Obter posição atualizada após compra
	assetPosition, _, err := broker.Position(ctx, pair)
	if err != nil {
		bot.DefaultLog.Error(err)
		return
	}

	// Criar mapa para ordens deste par se não existir
	if _, exists := t.activeOrders[pair]; !exists {
		t.activeOrders[pair] = make(map[int]int64)
	}

	// Se a saída parcial estiver habilitada, inicializar estruturas
	if t.config.UsePartialTakeProfit {
		// Limpar qualquer dado antigo
		t.partialPositions[pair] = make([]PartialPosition, 0)
		t.partialOrders[pair] = make([]int64, 0)

		// As posições parciais serão configuradas na próxima execução de checkPartialExits
	} else {
		// Definir take profit e stop loss com ordem OCO
		takeProfitPrice := closePrice * (1.0 + t.calculateTakeProfit(df, pair))

		orders, err := broker.CreateOrderOCO(
			ctx,
			core.SideTypeSell,
			pair,
			assetPosition,
			takeProfitPrice,
			stopLossPrice,
			stopLossPrice,
		)

		if err != nil {
			bot.DefaultLog.WithFields(map[string]any{
				"pair":       pair,
				"side":       core.SideTypeSell,
				"asset":      assetPosition,
				"takeProfit": takeProfitPrice,
				"stopPrice":  stopLossPrice,
			}).Error(err)
		} else if len(orders) > 0 {
			// Armazenar IDs das ordens para referência futura
			for i, order := range orders {
				t.activeOrders[pair][i] = order.ID
			}
		}
	}

	bot.DefaultLog.WithFields(map[string]any{
		"pair":              pair,
		"entryPrice":        closePrice,
		"positionSize":      positionSize,
		"stopLossPrice":     stopLossPrice,
		"takeProfitPrice":   closePrice * (1.0 + t.calculateTakeProfit(df, pair)),
		"consecutiveLosses": t.consecutiveLosses,
	}).Info("Entry executed")
}

// calculateTakeProfit calcula o objetivo de lucro com base nas configurações
func (t *TrendMaster) calculateTakeProfit(df *core.Dataframe, pair string) float64 {
	// Se alvos dinâmicos não estiverem habilitados, usar alvo base
	if !t.config.UseDynamicTargets {
		return t.config.BaseTarget
	}

	// Calcular alvo baseado em ATR
	atr := df.Metadata["atr"].Last(0)
	closePrice := df.Close.Last(0)

	if atr <= 0 || closePrice <= 0 {
		return t.config.BaseTarget
	}

	// Calcular alvo dinâmico com base na volatilidade (ATR)
	atrPercent := atr / closePrice
	dynamicTarget := atrPercent * t.config.AtrTargetFactor

	// Limitar entre mínimo e máximo configurados
	if dynamicTarget < t.config.MinTarget {
		dynamicTarget = t.config.MinTarget
	} else if dynamicTarget > t.config.MaxTarget {
		dynamicTarget = t.config.MaxTarget
	}

	return dynamicTarget
}

// executeExit executa a operação de saída
func (t *TrendMaster) executeExit(ctx context.Context, df *core.Dataframe, broker core.Broker, assetPosition float64) {
	pair := df.Pair
	currentPrice := df.Close.Last(0)

	// Cancelar todas as ordens ativas
	if ordersMap, exists := t.activeOrders[pair]; exists {
		for _, orderID := range ordersMap {
			order, err := broker.Order(ctx, pair, orderID)
			if err == nil {
				err = broker.Cancel(ctx, order)
				if err != nil {
					bot.DefaultLog.WithFields(map[string]any{
						"pair":    pair,
						"orderID": orderID,
					}).Error(err)
				}
			}
		}
		delete(t.activeOrders, pair)
	}

	// Cancelar ordens de saída parcial
	if orders, exists := t.partialOrders[pair]; exists && len(orders) > 0 {
		for _, orderID := range orders {
			order, err := broker.Order(ctx, pair, orderID)
			if err == nil {
				_ = broker.Cancel(ctx, order)
			}
		}
		delete(t.partialOrders, pair)
	}

	// Vender posição inteira
	_, err := broker.CreateOrderMarket(ctx, core.SideTypeSell, pair, assetPosition)
	if err != nil {
		bot.DefaultLog.WithFields(map[string]any{
			"pair":  pair,
			"side":  core.SideTypeSell,
			"asset": assetPosition,
			"price": currentPrice,
		}).Error(err)
		return
	}

	// Verificar resultado da operação
	entryPrice, exists := t.entryPrice[pair]
	if exists {
		tradeProfit := currentPrice > entryPrice
		t.lastTradeResult[pair] = tradeProfit

		// Atualizar contadores de vitórias/derrotas
		if tradeProfit {
			t.winCount++
			t.winStreak++
			t.lossStreak = 0
			t.consecutiveLosses = 0 // Reiniciar após operação lucrativa
		} else {
			t.lossCount++
			t.lossStreak++
			t.winStreak = 0
			t.consecutiveLosses++ // Incrementar contador de perdas
		}

		profitPercent := (currentPrice - entryPrice) / entryPrice * 100

		bot.DefaultLog.WithFields(map[string]any{
			"pair":              pair,
			"entryPrice":        entryPrice,
			"exitPrice":         currentPrice,
			"profit":            profitPercent,
			"isProfit":          tradeProfit,
			"consecutiveLosses": t.consecutiveLosses,
			"winRate":           float64(t.winCount) / float64(t.winCount+t.lossCount) * 100,
		}).Info("Exit executed")

		// Limpar preço de entrada
		delete(t.entryPrice, pair)
	}

	// Limpar outros dados de rastreamento para este par
	delete(t.positionSize, pair)
	delete(t.partialPositions, pair)
}
