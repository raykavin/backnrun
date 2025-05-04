// Package exchange provides implementations for various data exchange mechanisms
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

	"github.com/raykavin/backnrun/core"
	"github.com/samber/lo"
	"github.com/xhit/go-str2duration/v2"
)

// ---------------------
// Constants and Errors
// ---------------------

var (
	// ErrInsufficientData is returned when there is not enough data to fulfill a request
	ErrInsufficientData = errors.New("insufficient data")

	// defaultHeaderMap defines the standard CSV column mapping
	defaultHeaderMap = map[string]int{
		"time": 0, "open": 1, "close": 2, "low": 3, "high": 4, "volume": 5,
	}
)

// ---------------------
// Types
// ---------------------

// PairFeed represents data for a specific trading pair
type PairFeed struct {
	Pair       string
	File       string
	Timeframe  string
	HeikinAshi bool
}

// CSVFeed represents a data feed from CSV files
type CSVFeed struct {
	Feeds               map[string]PairFeed
	CandlePairTimeFrame map[string][]core.Candle
}

// PeriodBoundaryCheck defines an interface for checking period boundaries
type PeriodBoundaryCheck func(t time.Time, fromTimeframe, targetTimeframe string) (bool, error)

// ---------------------
// Constructor
// ---------------------

// NewCSVFeed creates a new CSV feed and resamples data to the target timeframe
func NewCSVFeed(targetTimeframe string, feeds ...PairFeed) (*CSVFeed, error) {
	csvFeed := &CSVFeed{
		Feeds:               make(map[string]PairFeed),
		CandlePairTimeFrame: make(map[string][]core.Candle),
	}

	for _, feed := range feeds {
		csvFeed.Feeds[feed.Pair] = feed

		// Read candles from CSV file
		candles, err := readCandlesFromCSV(feed)
		if err != nil {
			return nil, err
		}

		// Store the original candles
		sourceKey := csvFeed.feedTimeframeKey(feed.Pair, feed.Timeframe)
		csvFeed.CandlePairTimeFrame[sourceKey] = candles

		// Resample to target timeframe if different
		if err := csvFeed.resample(feed.Pair, feed.Timeframe, targetTimeframe); err != nil {
			return nil, err
		}
	}

	return csvFeed, nil
}

// ---------------------
// CSV Processing
// ---------------------

// readCandlesFromCSV reads and processes a CSV file to create candles
func readCandlesFromCSV(feed PairFeed) ([]core.Candle, error) {
	// Open CSV file
	csvFile, err := os.Open(feed.File)
	if err != nil {
		return nil, err
	}
	defer csvFile.Close()

	// Read all lines from the CSV
	csvLines, err := csv.NewReader(csvFile).ReadAll()
	if err != nil {
		return nil, err
	}

	// Parse headers
	headerMap, additionalHeaders, hasCustomHeaders := parseHeaders(csvLines[0])
	if hasCustomHeaders {
		csvLines = csvLines[1:] // Remove header row
	}

	// Initialize HeikinAshi if needed
	ha := core.NewHeikinAshi()

	// Process each CSV line
	candles := make([]core.Candle, 0, len(csvLines))
	for _, line := range csvLines {
		candle, err := parseCandleFromLine(line, headerMap, additionalHeaders, hasCustomHeaders, feed.Pair)
		if err != nil {
			return nil, err
		}

		// Convert to HeikinAshi if needed
		if feed.HeikinAshi {
			candle = candle.ToHeikinAshi(ha)
		}

		candles = append(candles, candle)
	}

	return candles, nil
}

// parseHeaders analyzes CSV headers and returns an index map
func parseHeaders(headers []string) (headerMap map[string]int, additional []string, hasCustomHeaders bool) {
	// Check if first element is a number (not a header)
	if _, err := strconv.Atoi(headers[0]); err == nil {
		return defaultHeaderMap, nil, false
	}

	// Initialize header map
	headerMap = make(map[string]int)

	// Process each header
	for index, header := range headers {
		headerMap[header] = index

		// Check if it's an additional header not in defaults
		if _, exists := defaultHeaderMap[header]; !exists {
			additional = append(additional, header)
		}
	}

	return headerMap, additional, true
}

// parseCandleFromLine parses a CSV line and creates a candle
func parseCandleFromLine(line []string, headerMap map[string]int, additionalHeaders []string, hasCustomHeaders bool, pair string) (core.Candle, error) {
	// Process timestamp
	timestamp, err := strconv.Atoi(line[headerMap["time"]])
	if err != nil {
		return core.Candle{}, err
	}

	// Create basic candle
	candle := core.Candle{
		Time:      time.Unix(int64(timestamp), 0).UTC(),
		UpdatedAt: time.Unix(int64(timestamp), 0).UTC(),
		Pair:      pair,
		Complete:  true,
	}

	// Process OHLCV values
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

	// Process additional metadata if present
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

// ---------------------
// Timeframe Handling
// ---------------------

// isFistCandlePeriod checks if a candle is the first in a period
func isFistCandlePeriod(t time.Time, fromTimeframe, targetTimeframe string) (bool, error) {
	fromDuration, err := str2duration.ParseDuration(fromTimeframe)
	if err != nil {
		return false, err
	}

	prev := t.Add(-fromDuration).UTC()
	return isLastCandlePeriod(prev, fromTimeframe, targetTimeframe)
}

// isLastCandlePeriod checks if a candle is the last in a period
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

// isTimeOnPeriodBoundary checks if a timestamp is on a period boundary
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

// ---------------------
// Resampling
// ---------------------

// resample resamples candles from source timeframe to target timeframe
func (c *CSVFeed) resample(pair, sourceTimeframe, targetTimeframe string) error {
	sourceKey := c.feedTimeframeKey(pair, sourceTimeframe)
	targetKey := c.feedTimeframeKey(pair, targetTimeframe)

	sourceCandles := c.CandlePairTimeFrame[sourceKey]
	if len(sourceCandles) == 0 {
		return nil
	}

	// Find the index of the first candle that starts a period
	startIdx, err := c.findFirstPeriodCandle(sourceCandles, sourceTimeframe, targetTimeframe)
	if err != nil {
		return err
	}

	// Perform resampling
	targetCandles, err := c.resampleCandles(sourceCandles[startIdx:], sourceTimeframe, targetTimeframe)
	if err != nil {
		return err
	}

	c.CandlePairTimeFrame[targetKey] = targetCandles
	return nil
}

// findFirstPeriodCandle finds the index of the first candle that starts a period
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
	return 0, nil // If not found, start from beginning
}

// resampleCandles resamples candles by grouping them by period
func (c *CSVFeed) resampleCandles(sourceCandles []core.Candle, sourceTimeframe, targetTimeframe string) ([]core.Candle, error) {
	if len(sourceCandles) == 0 {
		return nil, nil
	}

	targetCandles := make([]core.Candle, 0, len(sourceCandles)/4) // Initial size estimate

	var currentCandle core.Candle
	inPeriod := false

	for _, candle := range sourceCandles {
		isLast, err := isLastCandlePeriod(candle.Time, sourceTimeframe, targetTimeframe)
		if err != nil {
			return nil, err
		}

		// If not in a period, start a new one
		if !inPeriod {
			currentCandle = candle
			inPeriod = true
			continue
		}

		// Update current candle with data from current candle
		currentCandle.High = math.Max(currentCandle.High, candle.High)
		currentCandle.Low = math.Min(currentCandle.Low, candle.Low)
		currentCandle.Close = candle.Close
		currentCandle.Volume += candle.Volume

		// If this is the last candle of the period, finalize and add to list
		if isLast {
			currentCandle.Complete = true
			targetCandles = append(targetCandles, currentCandle)
			inPeriod = false
		}
	}

	// If the last period wasn't complete, only include if it's marked complete
	if inPeriod && !currentCandle.Complete {
		// Don't add
	} else if inPeriod {
		targetCandles = append(targetCandles, currentCandle)
	}

	return targetCandles, nil
}

// ---------------------
// Utility Methods
// ---------------------

// feedTimeframeKey generates a unique key for each pair and timeframe
func (c CSVFeed) feedTimeframeKey(pair, timeframe string) string {
	return fmt.Sprintf("%s--%s", pair, timeframe)
}

// Limit limits candles to a specific time duration
func (c *CSVFeed) Limit(duration time.Duration) *CSVFeed {
	for pair, candles := range c.CandlePairTimeFrame {
		if len(candles) == 0 {
			continue
		}

		// Calculate period start
		start := candles[len(candles)-1].Time.Add(-duration)

		// Filter candles to keep only those within the period
		c.CandlePairTimeFrame[pair] = lo.Filter(candles, func(candle core.Candle, _ int) bool {
			return candle.Time.After(start)
		})
	}
	return c
}

// ---------------------
// API Methods
// ---------------------

// AssetsInfo returns information about a trading pair's assets
func (c CSVFeed) AssetsInfo(pair string) (core.AssetInfo, error) {
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

// LastQuote returns the last quote (not implemented for CSVFeed)
func (c CSVFeed) LastQuote(_ context.Context, _ string) (float64, error) {
	return 0, errors.New("invalid operation")
}

// CandlesByPeriod returns candles within a specific time period
func (c CSVFeed) CandlesByPeriod(_ context.Context, pair, timeframe string, start, end time.Time) ([]core.Candle, error) {
	key := c.feedTimeframeKey(pair, timeframe)
	result := make([]core.Candle, 0)

	// Filter candles by period
	for _, candle := range c.CandlePairTimeFrame[key] {
		if candle.Time.Before(start) || candle.Time.After(end) {
			continue
		}
		result = append(result, candle)
	}

	return result, nil
}

// CandlesByLimit returns a limited number of candles and removes them from the feed
func (c *CSVFeed) CandlesByLimit(_ context.Context, pair, timeframe string, limit int) ([]core.Candle, error) {
	key := c.feedTimeframeKey(pair, timeframe)

	if len(c.CandlePairTimeFrame[key]) < limit {
		return nil, fmt.Errorf("%w: %s", ErrInsufficientData, pair)
	}

	// Get candles and update feed
	result := c.CandlePairTimeFrame[key][:limit]
	c.CandlePairTimeFrame[key] = c.CandlePairTimeFrame[key][limit:]

	return result, nil
}

// CandlesSubscription returns a channel to receive candles
func (c CSVFeed) CandlesSubscription(_ context.Context, pair, timeframe string) (chan core.Candle, chan error) {
	ccandle := make(chan core.Candle)
	cerr := make(chan error)
	key := c.feedTimeframeKey(pair, timeframe)

	go func() {
		defer close(ccandle)
		defer close(cerr)

		// Send all candles through the channel
		for _, candle := range c.CandlePairTimeFrame[key] {
			ccandle <- candle
		}
	}()

	return ccandle, cerr
}
