package strategies

import (
	"context"

	"github.com/raykavin/backnrun/bot"
	"github.com/raykavin/backnrun/core"
	"github.com/raykavin/backnrun/indicator"
)

// MACDDivergenceStrategy implementa uma estratégia baseada em divergências
// entre o preço e o MACD para identificar potenciais reversões
type MACDDivergenceStrategy struct {
	// Parâmetros do MACD
	fastPeriod   int
	slowPeriod   int
	signalPeriod int

	// Parâmetros da divergência
	lookbackPeriod int // Período para verificar divergências

	// Parâmetros de posição
	positionSize float64
	stopLoss     float64
	takeProfit   float64

	// Rastreamento interno
	activeOrders map[string]int64 // Rastreia ordens stop ativas
}

// NewMACDDivergenceStrategy cria uma nova instância da estratégia com parâmetros padrão
func NewMACDDivergenceStrategy() *MACDDivergenceStrategy {
	return &MACDDivergenceStrategy{
		fastPeriod:     12,
		slowPeriod:     26,
		signalPeriod:   9,
		lookbackPeriod: 14,
		positionSize:   0.5,  // 50% do capital disponível
		stopLoss:       0.03, // 3% de stop loss
		takeProfit:     0.06, // 6% de take profit
		activeOrders:   make(map[string]int64),
	}
}

// Timeframe retorna o timeframe necessário para esta estratégia
func (m MACDDivergenceStrategy) Timeframe() string {
	return "4h" // Timeframe de 4 horas
}

// WarmupPeriod retorna o número de candles necessários antes da estratégia estar pronta
func (m MACDDivergenceStrategy) WarmupPeriod() int {
	// MACD precisa do período maior (slow) + signal + lookback para divergências
	return m.slowPeriod + m.signalPeriod + m.lookbackPeriod
}

// Indicators calcula e retorna os indicadores usados por esta estratégia
func (m MACDDivergenceStrategy) Indicators(df *core.Dataframe) []core.ChartIndicator {
	// Calcular MACD
	df.Metadata["macd"], df.Metadata["macd_signal"], df.Metadata["macd_hist"] = indicator.MACD(
		df.Close,
		m.fastPeriod,
		m.slowPeriod,
		m.signalPeriod,
	)

	// Retornar indicadores para visualização
	return []core.ChartIndicator{
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
	}
}

// OnCandle é chamado para cada novo candle e implementa a lógica de trading
func (m *MACDDivergenceStrategy) OnCandle(ctx context.Context, df *core.Dataframe, broker core.Broker) {
	pair := df.Pair
	closePrice := df.Close.Last(0)

	// Obter posição atual
	assetPosition, quotePosition, err := broker.Position(ctx, pair)
	if err != nil {
		bot.DefaultLog.Error(err)
		return
	}

	// Verificar divergências
	bullishDivergence, bearishDivergence := m.checkDivergences(df)

	// Verificar sinais de entrada e saída
	if assetPosition > 0 {
		// Já estamos em posição longa, verificar saída
		if bearishDivergence && m.isMACDCrossingDown(df) {
			m.executeExit(ctx, df, broker, assetPosition)
		}
	} else {
		// Sem posição, verificar entrada
		if bullishDivergence && m.isMACDCrossingUp(df) {
			m.executeBuy(ctx, df, broker, quotePosition, closePrice)
		}
	}
}

// checkDivergences verifica padrões de divergência entre preço e MACD
func (m *MACDDivergenceStrategy) checkDivergences(df *core.Dataframe) (bullish bool, bearish bool) {
	// Inicializar valores
	bullish = false
	bearish = false

	// Encontrar mínimos e máximos locais
	priceHighs, priceLows := m.findLocalExtremes(df.Close, m.lookbackPeriod)
	macdHighs, macdLows := m.findLocalExtremes(df.Metadata["macd"], m.lookbackPeriod)

	// Verificar divergência de baixa (bearish): preço faz máximos mais altos, MACD faz máximos mais baixos
	if len(priceHighs) >= 2 && len(macdHighs) >= 2 {
		// Selecionar os dois últimos máximos
		lastPriceHigh := priceHighs[len(priceHighs)-1]
		prevPriceHigh := priceHighs[len(priceHighs)-2]
		lastMACDHigh := macdHighs[len(macdHighs)-1]
		prevMACDHigh := macdHighs[len(macdHighs)-2]

		// Verificar se preço subiu mas MACD caiu
		if lastPriceHigh > prevPriceHigh && lastMACDHigh < prevMACDHigh {
			bearish = true
		}
	}

	// Verificar divergência de alta (bullish): preço faz mínimos mais baixos, MACD faz mínimos mais altos
	if len(priceLows) >= 2 && len(macdLows) >= 2 {
		// Selecionar os dois últimos mínimos
		lastPriceLow := priceLows[len(priceLows)-1]
		prevPriceLow := priceLows[len(priceLows)-2]
		lastMACDLow := macdLows[len(macdLows)-1]
		prevMACDLow := macdLows[len(macdLows)-2]

		// Verificar se preço caiu mas MACD subiu
		if lastPriceLow < prevPriceLow && lastMACDLow > prevMACDLow {
			bullish = true
		}
	}

	return bullish, bearish
}

// findLocalExtremes encontra máximos e mínimos locais em uma série
func (m *MACDDivergenceStrategy) findLocalExtremes(
	series core.Series[float64],
	lookback int,
) (highs []float64, lows []float64) {
	highs = []float64{}
	lows = []float64{}

	// Precisamos de pelo menos 3 pontos para encontrar extremos
	if series.Length() < 3 {
		return highs, lows
	}

	// Limite o lookback ao tamanho da série
	if lookback > series.Length() {
		lookback = series.Length()
	}

	// Analisar a série, ignorando o último ponto (atual)
	for i := 2; i < lookback; i++ {
		// Valores para comparação
		prev := series.Last(i + 1)
		curr := series.Last(i)
		next := series.Last(i - 1)

		// Máximo local
		if curr > prev && curr > next {
			highs = append(highs, curr)
		}

		// Mínimo local
		if curr < prev && curr < next {
			lows = append(lows, curr)
		}
	}

	return highs, lows
}

// isMACDCrossingUp verifica se o MACD está cruzando para cima da linha de sinal
func (m *MACDDivergenceStrategy) isMACDCrossingUp(df *core.Dataframe) bool {
	macd := df.Metadata["macd"]
	signal := df.Metadata["macd_signal"]

	return macd.Last(1) <= signal.Last(1) && macd.Last(0) > signal.Last(0)
}

// isMACDCrossingDown verifica se o MACD está cruzando para baixo da linha de sinal
func (m *MACDDivergenceStrategy) isMACDCrossingDown(df *core.Dataframe) bool {
	macd := df.Metadata["macd"]
	signal := df.Metadata["macd_signal"]

	return macd.Last(1) >= signal.Last(1) && macd.Last(0) < signal.Last(0)
}

// executeExit executa a operação de saída
func (m *MACDDivergenceStrategy) executeExit(ctx context.Context, df *core.Dataframe, broker core.Broker, assetPosition float64) {
	pair := df.Pair

	// Cancelar qualquer ordem OCO ativa
	if orderID, exists := m.activeOrders[pair]; exists {
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
		// Remover a referência à ordem
		delete(m.activeOrders, pair)
	}

	// Vender toda a posição
	_, err := broker.CreateOrderMarket(
		ctx,
		core.SideTypeSell,
		pair,
		assetPosition,
	)

	if err != nil {
		bot.DefaultLog.WithFields(map[string]any{
			"pair":  pair,
			"side":  core.SideTypeSell,
			"asset": assetPosition,
			"price": df.Close.Last(0),
		}).Error(err)
	}
}

// executeBuy executa a operação de compra
func (m *MACDDivergenceStrategy) executeBuy(ctx context.Context, df *core.Dataframe, broker core.Broker, quotePosition, closePrice float64) {
	pair := df.Pair

	// Calcular tamanho da posição baseado no capital disponível
	entryAmount := quotePosition * m.positionSize

	// Executar ordem de compra a mercado
	_, err := broker.CreateOrderMarketQuote(
		ctx,
		core.SideTypeBuy,
		pair,
		entryAmount,
	)

	if err != nil {
		bot.DefaultLog.WithFields(map[string]any{
			"pair":  pair,
			"side":  core.SideTypeBuy,
			"quote": entryAmount,
			"price": closePrice,
		}).Error(err)
		return
	}

	// Obter posição atualizada após a compra
	assetPosition, _, err := broker.Position(ctx, pair)
	if err != nil {
		bot.DefaultLog.Error(err)
		return
	}

	// Usar OCO order para configurar tanto stop loss quanto take profit
	stopPrice := closePrice * (1.0 - m.stopLoss)
	takeProfitPrice := closePrice * (1.0 + m.takeProfit)

	orders, err := broker.CreateOrderOCO(
		ctx,
		core.SideTypeSell,
		pair,
		assetPosition,
		takeProfitPrice,
		stopPrice,
		stopPrice,
	)

	if err != nil {
		bot.DefaultLog.WithFields(map[string]any{
			"pair":       pair,
			"side":       core.SideTypeSell,
			"asset":      assetPosition,
			"takeProfit": takeProfitPrice,
			"stopPrice":  stopPrice,
		}).Error(err)
	} else if len(orders) > 0 {
		// Armazenar ID da primeira ordem para referência futura
		m.activeOrders[pair] = orders[0].ID
	}
}
