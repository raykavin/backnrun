package strategies

import (
	"github.com/raykavin/backnrun"
	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/indicator"
	"github.com/raykavin/backnrun/pkg/strategy"
)

// TripleMAStrategy implementa uma estratégia de cruzamento de três médias móveis
// para identificar tendências fortes e pontos de entrada/saída
type TripleMAStrategy struct {
	// Parâmetros das Médias Móveis
	shortPeriod  int
	mediumPeriod int
	longPeriod   int

	// Parâmetros de posição
	positionSize float64
	stopLoss     float64 // Percentual de stop loss

	// Rastreamento interno
	activeOrders map[string]int64 // Rastreia ordens stop ativas
}

// NewTripleMAStrategy cria uma nova instância da estratégia com parâmetros padrão
func NewTripleMAStrategy() *TripleMAStrategy {
	return &TripleMAStrategy{
		shortPeriod:  9,
		mediumPeriod: 21,
		longPeriod:   50,
		positionSize: 0.5,  // 50% do capital disponível
		stopLoss:     0.05, // 5% do preço de entrada
		activeOrders: make(map[string]int64),
	}
}

// Timeframe retorna o timeframe necessário para esta estratégia
func (t TripleMAStrategy) Timeframe() string {
	return "1h" // Timeframe de 1 hora
}

// WarmupPeriod retorna o número de candles necessários antes da estratégia estar pronta
func (t TripleMAStrategy) WarmupPeriod() int {
	// Usar o período mais longo das três médias + margem adicional
	return t.longPeriod + 10
}

// Indicators calcula e retorna os indicadores usados por esta estratégia
func (t TripleMAStrategy) Indicators(df *core.Dataframe) []strategy.ChartIndicator {
	// Calcular as três médias móveis
	df.Metadata["short_ma"] = indicator.EMA(df.Close, t.shortPeriod)
	df.Metadata["medium_ma"] = indicator.EMA(df.Close, t.mediumPeriod)
	df.Metadata["long_ma"] = indicator.EMA(df.Close, t.longPeriod)

	// Retornar indicadores para visualização
	return []strategy.ChartIndicator{
		{
			Overlay:   true,
			GroupName: "Moving Averages",
			Time:      df.Time,
			Metrics: []strategy.IndicatorMetric{
				{
					Values: df.Metadata["short_ma"],
					Name:   "Short EMA",
					Color:  "blue",
					Style:  strategy.StyleLine,
				},
				{
					Values: df.Metadata["medium_ma"],
					Name:   "Medium EMA",
					Color:  "green",
					Style:  strategy.StyleLine,
				},
				{
					Values: df.Metadata["long_ma"],
					Name:   "Long EMA",
					Color:  "red",
					Style:  strategy.StyleLine,
				},
			},
		},
	}
}

// OnCandle é chamado para cada novo candle e implementa a lógica de trading
func (t *TripleMAStrategy) OnCandle(df *core.Dataframe, broker core.Broker) {
	pair := df.Pair
	closePrice := df.Close.Last(0)

	// Obter valores das médias móveis
	shortMA := df.Metadata["short_ma"].Last(0)
	mediumMA := df.Metadata["medium_ma"].Last(0)
	longMA := df.Metadata["long_ma"].Last(0)

	// Valores anteriores para detectar cruzamentos
	prevShortMA := df.Metadata["short_ma"].Last(1)
	prevMediumMA := df.Metadata["medium_ma"].Last(1)

	// Obter posição atual
	assetPosition, quotePosition, err := broker.Position(pair)
	if err != nil {
		backnrun.DefaultLog.Error(err)
		return
	}

	// Verificar sinais de entrada e saída
	if assetPosition > 0 {
		// Já estamos em posição, verificar saída
		if t.shouldExit(shortMA, mediumMA, prevShortMA, prevMediumMA) {
			t.executeExit(df, broker, assetPosition)
		}
	} else {
		// Sem posição, verificar entrada
		if t.shouldEnter(shortMA, mediumMA, longMA, prevShortMA, prevMediumMA) {
			t.executeEntry(df, broker, quotePosition, closePrice)
		}
	}
}

// shouldEnter verifica se as condições de entrada são atendidas
func (t *TripleMAStrategy) shouldEnter(shortMA, mediumMA, longMA, prevShortMA, prevMediumMA float64) bool {
	// Entrar quando a média curta cruza acima da média média
	// E ambas estão acima da média longa (tendência de alta)
	crossedAbove := prevShortMA <= prevMediumMA && shortMA > mediumMA
	allAligned := shortMA > mediumMA && mediumMA > longMA

	return crossedAbove && allAligned
}

// shouldExit verifica se as condições de saída são atendidas
func (t *TripleMAStrategy) shouldExit(shortMA, mediumMA, prevShortMA, prevMediumMA float64) bool {
	// Sair quando a média curta cruza abaixo da média média
	return prevShortMA >= prevMediumMA && shortMA < mediumMA
}

// executeEntry executa a operação de entrada
func (t *TripleMAStrategy) executeEntry(df *core.Dataframe, broker core.Broker, quotePosition, closePrice float64) {
	pair := df.Pair

	// Calcular tamanho da posição baseado no capital disponível
	entryAmount := quotePosition * t.positionSize

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

	// Obter posição atualizada após a compra
	assetPosition, _, err := broker.Position(pair)
	if err != nil {
		backnrun.DefaultLog.Error(err)
		return
	}

	// Criar ordem de stop loss
	stopPrice := closePrice * (1.0 - t.stopLoss)
	stopOrder, err := broker.CreateOrderStop(pair, assetPosition, stopPrice)
	if err != nil {
		backnrun.DefaultLog.WithFields(map[string]interface{}{
			"pair":      pair,
			"asset":     assetPosition,
			"stopPrice": stopPrice,
		}).Error(err)
	} else {
		// Armazenar ID da ordem stop para referência futura
		t.activeOrders[pair] = stopOrder.ID
	}
}

// executeExit executa a operação de saída
func (t *TripleMAStrategy) executeExit(df *core.Dataframe, broker core.Broker, assetPosition float64) {
	pair := df.Pair

	// Cancelar a ordem de stop loss se estiver ativa
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
			"price": df.Close.Last(0),
		}).Error(err)
	}
}
