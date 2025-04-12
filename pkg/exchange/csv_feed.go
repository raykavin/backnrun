package exchange

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/raykavin/backnrun/pkg/core"
	"github.com/samber/lo"
	"github.com/xhit/go-str2duration/v2"
)

var (
	ErrInsufficientData = errors.New("insufficient data")
	defaultHeaderMap    = map[string]int{
		"time": 0, "open": 1, "close": 2, "low": 3, "high": 4, "volume": 5,
	}
)

// PairFeed representa os dados de um par no feed de dados
type PairFeed struct {
	Pair       string
	File       string
	Timeframe  string
	HeikinAshi bool
}

// CSVFeed representa um feed de dados de CSV
type CSVFeed struct {
	Feeds               map[string]PairFeed
	CandlePairTimeFrame map[string][]core.Candle
}

// AssetsInfo retorna informações sobre os ativos de um par
func (c CSVFeed) AssetsInfo(pair string) core.AssetInfo {
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

// parseHeaders analisa os cabeçalhos do CSV e retorna um mapa de índices
func parseHeaders(headers []string) (headerMap map[string]int, additional []string, hasCustomHeaders bool) {
	// Verifica se o primeiro elemento é um número (não é cabeçalho)
	if _, err := strconv.Atoi(headers[0]); err == nil {
		return defaultHeaderMap, nil, false
	}

	// Inicializa o mapa de cabeçalhos
	headerMap = make(map[string]int)

	// Processa cada cabeçalho
	for index, header := range headers {
		headerMap[header] = index

		// Verifica se é um cabeçalho adicional que não está nos padrões
		if _, exists := defaultHeaderMap[header]; !exists {
			additional = append(additional, header)
		}
	}

	return headerMap, additional, true
}

// NewCSVFeed cria um novo feed de dados de CSV e faz o resample para o timeframe alvo
func NewCSVFeed(targetTimeframe string, feeds ...PairFeed) (*CSVFeed, error) {
	csvFeed := &CSVFeed{
		Feeds:               make(map[string]PairFeed),
		CandlePairTimeFrame: make(map[string][]core.Candle),
	}

	for _, feed := range feeds {
		csvFeed.Feeds[feed.Pair] = feed

		// Lê o arquivo CSV
		candles, err := readCandlesFromCSV(feed)
		if err != nil {
			return nil, err
		}

		// Armazena os candles no mapa
		sourceKey := csvFeed.feedTimeframeKey(feed.Pair, feed.Timeframe)
		csvFeed.CandlePairTimeFrame[sourceKey] = candles

		// Faz o resample para o timeframe alvo
		if err := csvFeed.resample(feed.Pair, feed.Timeframe, targetTimeframe); err != nil {
			return nil, err
		}
	}

	return csvFeed, nil
}

// readCandlesFromCSV lê e processa o arquivo CSV para criar candles
func readCandlesFromCSV(feed PairFeed) ([]core.Candle, error) {
	// Abre o arquivo CSV
	csvFile, err := os.Open(feed.File)
	if err != nil {
		return nil, err
	}
	defer csvFile.Close()

	// Lê todas as linhas do CSV
	csvLines, err := csv.NewReader(csvFile).ReadAll()
	if err != nil {
		return nil, err
	}

	// Analisa os cabeçalhos
	headerMap, additionalHeaders, hasCustomHeaders := parseHeaders(csvLines[0])
	if hasCustomHeaders {
		csvLines = csvLines[1:] // Remove a linha de cabeçalho
	}

	// Inicializa o objeto HeikinAshi se necessário
	ha := core.NewHeikinAshi()

	// Processa cada linha do CSV
	candles := make([]core.Candle, 0, len(csvLines))
	for _, line := range csvLines {
		candle, err := parseCandleFromLine(line, headerMap, additionalHeaders, hasCustomHeaders, feed.Pair)
		if err != nil {
			return nil, err
		}

		// Converte para HeikinAshi se necessário
		if feed.HeikinAshi {
			candle = candle.ToHeikinAshi(ha)
		}

		candles = append(candles, candle)
	}

	return candles, nil
}

// parseCandleFromLine analisa uma linha do CSV e cria um candle
func parseCandleFromLine(line []string, headerMap map[string]int, additionalHeaders []string, hasCustomHeaders bool, pair string) (core.Candle, error) {
	// Processa o timestamp
	timestamp, err := strconv.Atoi(line[headerMap["time"]])
	if err != nil {
		return core.Candle{}, err
	}

	// Cria o candle básico
	candle := core.Candle{
		Time:      time.Unix(int64(timestamp), 0).UTC(),
		UpdatedAt: time.Unix(int64(timestamp), 0).UTC(),
		Pair:      pair,
		Complete:  true,
	}

	// Processa os valores OHLCV
	if candle.Open, err = strconv.ParseFloat(line[headerMap["open"]], 64); err != nil {
		return core.Candle{}, err
	}

	if candle.Close, err = strconv.ParseFloat(line[headerMap["close"]], 64); err != nil {
		return core.Candle{}, err
	}

	if candle.Low, err = strconv.ParseFloat(line[headerMap["low"]], 64); err != nil {
		return core.Candle{}, err
	}

	if candle.High, err = strconv.ParseFloat(line[headerMap["high"]], 64); err != nil {
		return core.Candle{}, err
	}

	if candle.Volume, err = strconv.ParseFloat(line[headerMap["volume"]], 64); err != nil {
		return core.Candle{}, err
	}

	// Processa metadados adicionais se existirem
	if hasCustomHeaders {
		candle.Metadata = make(map[string]float64, len(additionalHeaders))
		for _, header := range additionalHeaders {
			value, err := strconv.ParseFloat(line[headerMap[header]], 64)
			if err != nil {
				return core.Candle{}, err
			}
			candle.Metadata[header] = value
		}
	}

	return candle, nil
}

// feedTimeframeKey gera uma chave única para cada par e timeframe
func (c CSVFeed) feedTimeframeKey(pair, timeframe string) string {
	return fmt.Sprintf("%s--%s", pair, timeframe)
}

// LastQuote retorna a última cotação (não implementada para CSVFeed)
func (c CSVFeed) LastQuote(_ context.Context, _ string) (float64, error) {
	return 0, errors.New("invalid operation")
}

// Limit limita os candles a um período de tempo específico
func (c *CSVFeed) Limit(duration time.Duration) *CSVFeed {
	for pair, candles := range c.CandlePairTimeFrame {
		if len(candles) == 0 {
			continue
		}

		// Calcula o início do período
		start := candles[len(candles)-1].Time.Add(-duration)

		// Filtra os candles para manter apenas os que estão dentro do período
		c.CandlePairTimeFrame[pair] = lo.Filter(candles, func(candle core.Candle, _ int) bool {
			return candle.Time.After(start)
		})
	}
	return c
}

// PeriodBoundaryCheck define uma interface para verificar limites de períodos
type PeriodBoundaryCheck func(t time.Time, fromTimeframe, targetTimeframe string) (bool, error)

// isFistCandlePeriod verifica se um candle é o primeiro de um período
func isFistCandlePeriod(t time.Time, fromTimeframe, targetTimeframe string) (bool, error) {
	fromDuration, err := str2duration.ParseDuration(fromTimeframe)
	if err != nil {
		return false, err
	}

	prev := t.Add(-fromDuration).UTC()
	return isLastCandlePeriod(prev, fromTimeframe, targetTimeframe)
}

// isLastCandlePeriod verifica se um candle é o último de um período
func isLastCandlePeriod(t time.Time, fromTimeframe, targetTimeframe string) (bool, error) {
	if fromTimeframe == targetTimeframe {
		return true, nil
	}

	fromDuration, err := str2duration.ParseDuration(fromTimeframe)
	if err != nil {
		return false, err
	}

	next := t.Add(fromDuration).UTC()
	return isTimeOnPeriodBoundary(next, targetTimeframe)
}

// isTimeOnPeriodBoundary verifica se um timestamp está na fronteira de um período
func isTimeOnPeriodBoundary(t time.Time, targetTimeframe string) (bool, error) {
	switch targetTimeframe {
	case "1m":
		return t.Second() == 0, nil
	case "5m":
		return t.Minute()%5 == 0 && t.Second() == 0, nil
	case "10m":
		return t.Minute()%10 == 0 && t.Second() == 0, nil
	case "15m":
		return t.Minute()%15 == 0 && t.Second() == 0, nil
	case "30m":
		return t.Minute()%30 == 0 && t.Second() == 0, nil
	case "1h":
		return t.Minute() == 0 && t.Second() == 0, nil
	case "2h":
		return t.Hour()%2 == 0 && t.Minute() == 0 && t.Second() == 0, nil
	case "4h":
		return t.Hour()%4 == 0 && t.Minute() == 0 && t.Second() == 0, nil
	case "12h":
		return t.Hour()%12 == 0 && t.Minute() == 0 && t.Second() == 0, nil
	case "1d":
		return t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0, nil
	case "1w":
		return t.Weekday() == time.Sunday && t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0, nil
	default:
		return false, fmt.Errorf("invalid timeframe: %s", targetTimeframe)
	}
}

// resample faz o resampling dos candles de um timeframe para outro
func (c *CSVFeed) resample(pair, sourceTimeframe, targetTimeframe string) error {
	sourceKey := c.feedTimeframeKey(pair, sourceTimeframe)
	targetKey := c.feedTimeframeKey(pair, targetTimeframe)

	sourceCandles := c.CandlePairTimeFrame[sourceKey]
	if len(sourceCandles) == 0 {
		return nil
	}

	// Encontra o índice do primeiro candle que inicia um período
	startIdx, err := c.findFirstPeriodCandle(sourceCandles, sourceTimeframe, targetTimeframe)
	if err != nil {
		return err
	}

	// Faz o resampling
	targetCandles, err := c.resampleCandles(sourceCandles[startIdx:], sourceTimeframe, targetTimeframe)
	if err != nil {
		return err
	}

	c.CandlePairTimeFrame[targetKey] = targetCandles
	return nil
}

// findFirstPeriodCandle encontra o índice do primeiro candle que inicia um período
func (c *CSVFeed) findFirstPeriodCandle(candles []core.Candle, sourceTimeframe, targetTimeframe string) (int, error) {
	for i := range candles {
		isFirst, err := isFistCandlePeriod(candles[i].Time, sourceTimeframe, targetTimeframe)
		if err != nil {
			return 0, err
		}
		if isFirst {
			return i, nil
		}
	}
	return 0, nil // Se não encontrar, começa do início
}

// resampleCandles faz o resampling dos candles, agrupando-os por período
func (c *CSVFeed) resampleCandles(sourceCandles []core.Candle, sourceTimeframe, targetTimeframe string) ([]core.Candle, error) {
	if len(sourceCandles) == 0 {
		return nil, nil
	}

	targetCandles := make([]core.Candle, 0, len(sourceCandles)/4) // Estimativa inicial de tamanho

	var currentCandle core.Candle
	inPeriod := false

	for _, candle := range sourceCandles {
		isLast, err := isLastCandlePeriod(candle.Time, sourceTimeframe, targetTimeframe)
		if err != nil {
			return nil, err
		}

		// Se não estivermos em um período, começa um novo
		if !inPeriod {
			currentCandle = candle
			inPeriod = true
			continue
		}

		// Atualiza o candle atual com os dados do candle corrente
		currentCandle.High = math.Max(currentCandle.High, candle.High)
		currentCandle.Low = math.Min(currentCandle.Low, candle.Low)
		currentCandle.Close = candle.Close
		currentCandle.Volume += candle.Volume

		// Se este for o último candle do período, finaliza e adiciona à lista
		if isLast {
			currentCandle.Complete = true
			targetCandles = append(targetCandles, currentCandle)
			inPeriod = false
		}
	}

	// Se o último período não foi completo, não o inclui
	if inPeriod && !currentCandle.Complete {
		// Não adiciona
	} else if inPeriod {
		targetCandles = append(targetCandles, currentCandle)
	}

	return targetCandles, nil
}

// CandlesByPeriod retorna os candles dentro de um período específico
func (c CSVFeed) CandlesByPeriod(_ context.Context, pair, timeframe string, start, end time.Time) ([]core.Candle, error) {
	key := c.feedTimeframeKey(pair, timeframe)
	result := make([]core.Candle, 0)

	// Filtra os candles pelo período
	for _, candle := range c.CandlePairTimeFrame[key] {
		if candle.Time.Before(start) || candle.Time.After(end) {
			continue
		}
		result = append(result, candle)
	}

	return result, nil
}

// CandlesByLimit retorna um número limitado de candles e os remove do feed
func (c *CSVFeed) CandlesByLimit(_ context.Context, pair, timeframe string, limit int) ([]core.Candle, error) {
	key := c.feedTimeframeKey(pair, timeframe)

	if len(c.CandlePairTimeFrame[key]) < limit {
		return nil, fmt.Errorf("%w: %s", ErrInsufficientData, pair)
	}

	// Obtém os candles e atualiza o feed
	result := c.CandlePairTimeFrame[key][:limit]
	c.CandlePairTimeFrame[key] = c.CandlePairTimeFrame[key][limit:]

	return result, nil
}

// CandlesSubscription retorna um canal para receber candles
func (c CSVFeed) CandlesSubscription(_ context.Context, pair, timeframe string) (chan core.Candle, chan error) {
	ccandle := make(chan core.Candle)
	cerr := make(chan error)
	key := c.feedTimeframeKey(pair, timeframe)

	go func() {
		defer close(ccandle)
		defer close(cerr)

		// Envia todos os candles pelo canal
		for _, candle := range c.CandlePairTimeFrame[key] {
			ccandle <- candle
		}
	}()

	return ccandle, cerr
}
