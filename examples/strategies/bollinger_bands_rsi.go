package strategies

// import (
// 	"github.com/raykavin/backnrun/pkg/core"
// 	"github.com/raykavin/backnrun/pkg/indicator"
// 	"github.com/raykavin/backnrun/pkg/logger"
// 	"github.com/raykavin/backnrun/pkg/strategy"
// )

// // BollingerRSIStrategy combina Bollinger Bands para identificar volatilidade
// // e RSI para identificar condições de sobrecompra/sobrevenda
// type BollingerRSIStrategy struct {
// 	// Parâmetros das Bollinger Bands
// 	bollingerPeriod int
// 	bollingerDev    float64 // Desvio padrão (geralmente 2.0)

// 	// Parâmetros do RSI
// 	rsiPeriod     int
// 	rsiOverbought float64 // Nível de sobrecompra (geralmente 70)
// 	rsiOversold   float64 // Nível de sobrevenda (geralmente 30)

// 	// Parâmetros de posição
// 	positionSize float64
// 	trailingStop float64 // Percentual do trailing stop

// 	// Rastreamento interno
// 	activeOrders map[string]int64 // Rastreia ordens stop ativas
// }

// // NewBollingerRSIStrategy cria uma nova instância da estratégia com parâmetros padrão
// func NewBollingerRSIStrategy() *BollingerRSIStrategy {
// 	return &BollingerRSIStrategy{
// 		bollingerPeriod: 20,
// 		bollingerDev:    2.0,
// 		rsiPeriod:       14,
// 		rsiOverbought:   70.0,
// 		rsiOversold:     30.0,
// 		positionSize:    0.5,  // 50% do capital disponível
// 		trailingStop:    0.03, // 3% do preço
// 		activeOrders:    make(map[string]int64),
// 	}
// }

// // Timeframe retorna o timeframe necessário para esta estratégia
// func (b BollingerRSIStrategy) Timeframe() string {
// 	return "4h" // Timeframe de 4 horas
// }

// // WarmupPeriod retorna o número de candles necessários antes da estratégia estar pronta
// func (b BollingerRSIStrategy) WarmupPeriod() int {
// 	// Usar o maior entre o período do RSI e Bollinger
// 	if b.rsiPeriod > b.bollingerPeriod {
// 		return b.rsiPeriod + 10
// 	}
// 	return b.bollingerPeriod + 10
// }

// // Indicators calcula e retorna os indicadores usados por esta estratégia
// func (b BollingerRSIStrategy) Indicators(df *core.Dataframe) []strategy.ChartIndicator {
// 	// Calcular Bollinger Bands
// 	df.Metadata["bb_middle"] = indicator.SMA(df.Close, b.bollingerPeriod)
// 	df.Metadata["bb_stddev"] = indicator.StdDev(df.Close, b.bollingerPeriod)
// 	df.Metadata["bb_upper"] = indicator.Add(
// 		df.Metadata["bb_middle"],
// 		indicator.Multiply(df.Metadata["bb_stddev"], b.bollingerDev),
// 	)
// 	df.Metadata["bb_lower"] = indicator.Subtract(
// 		df.Metadata["bb_middle"],
// 		indicator.Multiply(df.Metadata["bb_stddev"], b.bollingerDev),
// 	)

// 	// Calcular RSI
// 	df.Metadata["rsi"] = indicator.RSI(df.Close, b.rsiPeriod)

// 	// Retornar indicadores para visualização
// 	return []strategy.ChartIndicator{
// 		{
// 			Overlay:   true,
// 			GroupName: "Bollinger Bands",
// 			Time:      df.Time,
// 			Metrics: []strategy.IndicatorMetric{
// 				{
// 					Values: df.Metadata["bb_upper"],
// 					Name:   "BB Upper",
// 					Color:  "red",
// 					Style:  strategy.StyleLine,
// 				},
// 				{
// 					Values: df.Metadata["bb_middle"],
// 					Name:   "BB Middle",
// 					Color:  "blue",
// 					Style:  strategy.StyleLine,
// 				},
// 				{
// 					Values: df.Metadata["bb_lower"],
// 					Name:   "BB Lower",
// 					Color:  "red",
// 					Style:  strategy.StyleLine,
// 				},
// 			},
// 		},
// 		{
// 			Overlay:   false,
// 			GroupName: "RSI",
// 			Time:      df.Time,
// 			Metrics: []strategy.IndicatorMetric{
// 				{
// 					Values: df.Metadata["rsi"],
// 					Name:   "RSI(" + string(rune(b.rsiPeriod+'0')) + ")",
// 					Color:  "purple",
// 					Style:  strategy.StyleLine,
// 				},
// 			},
// 		},
// 	}
// }

// // OnCandle é chamado para cada novo candle e implementa a lógica de trading
// func (b *BollingerRSIStrategy) OnCandle(df *core.Dataframe, broker core.Broker) {
// 	pair := df.Pair
// 	closePrice := df.Close.Last(0)
// 	rsi := df.Metadata["rsi"].Last(0)
// 	bbLower := df.Metadata["bb_lower"].Last(0)
// 	bbUpper := df.Metadata["bb_upper"].Last(0)

// 	// Obter posição atual
// 	assetPosition, quotePosition, err := broker.Position(pair)
// 	if err != nil {
// 		logger.Error(err)
// 		return
// 	}

// 	// Verificar sinais de entrada e saída
// 	if assetPosition > 0 {
// 		// Já estamos em posição, verificar saída
// 		if b.shouldExit(closePrice, rsi, bbUpper) {
// 			b.executeExit(df, broker, assetPosition)
// 		}
// 	} else {
// 		// Sem posição, verificar entrada
// 		if b.shouldEnter(closePrice, rsi, bbLower) {
// 			b.executeEntry(df, broker, quotePosition, closePrice)
// 		}
// 	}
// }

// // shouldEnter verifica se as condições de entrada são atendidas
// func (b *BollingerRSIStrategy) shouldEnter(closePrice, rsi, bbLower float64) bool {
// 	// Entrar quando o preço está próximo da banda inferior e o RSI indica sobrevenda
// 	return closePrice <= bbLower*1.01 && rsi <= b.rsiOversold
// }

// // shouldExit verifica se as condições de saída são atendidas
// func (b *BollingerRSIStrategy) shouldExit(closePrice, rsi, bbUpper float64) bool {
// 	// Sair quando o preço está próximo da banda superior ou o RSI indica sobrecompra
// 	return closePrice >= bbUpper*0.99 || rsi >= b.rsiOverbought
// }

// // executeEntry executa a operação de entrada
// func (b *BollingerRSIStrategy) executeEntry(df *core.Dataframe, broker core.Broker, quotePosition, closePrice float64) {
// 	pair := df.Pair

// 	// Calcular tamanho da posição baseado no capital disponível
// 	entryAmount := quotePosition * b.positionSize

// 	// Executar ordem de compra a mercado
// 	_, err := broker.CreateOrderMarketQuote(core.SideTypeBuy, pair, entryAmount)
// 	if err != nil {
// 		logger.WithFields(map[string]interface{}{
// 			"pair":  pair,
// 			"side":  core.SideTypeBuy,
// 			"quote": entryAmount,
// 			"price": closePrice,
// 		}).Error(err)
// 		return
// 	}

// 	// Obter posição atualizada após a compra
// 	assetPosition, _, err := broker.Position(pair)
// 	if err != nil {
// 		logger.Error(err)
// 		return
// 	}

// 	// Criar ordem de stop loss
// 	stopPrice := closePrice * (1.0 - b.trailingStop)
// 	stopOrder, err := broker.CreateOrderStop(pair, assetPosition, stopPrice)
// 	if err != nil {
// 		logger.WithFields(map[string]interface{}{
// 			"pair":      pair,
// 			"asset":     assetPosition,
// 			"stopPrice": stopPrice,
// 		}).Error(err)
// 	} else {
// 		// Armazenar ID da ordem stop para referência futura
// 		b.activeOrders[pair] = stopOrder.ID
// 	}
// }

// // executeExit executa a operação de saída
// func (b *BollingerRSIStrategy) executeExit(df *core.Dataframe, broker core.Broker, assetPosition float64) {
// 	pair := df.Pair

// 	// Cancelar a ordem de stop loss se estiver ativa
// 	if orderID, exists := b.activeOrders[pair]; exists {
// 		order, err := broker.Order(pair, orderID)
// 		if err == nil {
// 			err = broker.Cancel(order)
// 			if err != nil {
// 				logger.WithFields(map[string]interface{}{
// 					"pair":    pair,
// 					"orderID": orderID,
// 				}).Error(err)
// 			}
// 		}
// 		// Remover a referência à ordem
// 		delete(b.activeOrders, pair)
// 	}

// 	// Vender toda a posição
// 	_, err := broker.CreateOrderMarket(core.SideTypeSell, pair, assetPosition)
// 	if err != nil {
// 		logger.WithFields(map[string]interface{}{
// 			"pair":  pair,
// 			"side":  core.SideTypeSell,
// 			"asset": assetPosition,
// 			"price": df.Close.Last(0),
// 		}).Error(err)
// 	}
// }
