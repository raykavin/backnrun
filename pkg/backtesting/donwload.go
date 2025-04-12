package backtesting

import (
	"context"
	"encoding/csv"
	"os"
	"time"

	"github.com/raykavin/backnrun/pkg/core"
	"github.com/schollz/progressbar/v3"
	"github.com/xhit/go-str2duration/v2"

	"github.com/rodrigo-brito/ninjabot/tools/log"
)

const (
	batchSize = 500
)

// CSV header names
var csvHeaders = []string{"time", "open", "close", "low", "high", "volume"}

// Downloader facilitates downloading historical candle data from exchanges
type Downloader struct {
	exchange core.Feeder
}

// NewDownloader creates a new downloader instance with the provided exchange
func NewDownloader(exchange core.Feeder) Downloader {
	return Downloader{
		exchange: exchange,
	}
}

// Parameters defines the time range for data download
type Parameters struct {
	Start time.Time
	End   time.Time
}

// Option is a function type for configuring download parameters
type Option func(*Parameters)

// WithInterval sets specific start and end times for the download
func WithInterval(start, end time.Time) Option {
	return func(parameters *Parameters) {
		parameters.Start = start
		parameters.End = end
	}
}

// WithDays sets the download period to a specific number of days from now
func WithDays(days int) Option {
	return func(parameters *Parameters) {
		parameters.Start = time.Now().AddDate(0, 0, -days)
		parameters.End = time.Now()
	}
}

// calculateCandleCount determines the number of candles in the given timeframe
func calculateCandleCount(start, end time.Time, timeframe string) (int, time.Duration, error) {
	totalDuration := end.Sub(start)
	interval, err := str2duration.ParseDuration(timeframe)
	if err != nil {
		return 0, 0, err
	}
	return int(totalDuration / interval), interval, nil
}

// Download fetches candle data from the exchange and saves it to a CSV file
func (d Downloader) Download(ctx context.Context, pair, timeframe, outputPath string, options ...Option) error {
	recordFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer recordFile.Close()

	// Apply download parameters
	parameters := initializeParameters()
	for _, option := range options {
		option(parameters)
	}
	normalizeTimeParameters(parameters)

	// Calculate candle count and interval
	candleCount, interval, err := calculateCandleCount(parameters.Start, parameters.End, timeframe)
	if err != nil {
		return err
	}
	candleCount++

	log.Infof("Downloading %d candles of %s for %s", candleCount, timeframe, pair)

	// Setup CSV writer
	writer := csv.NewWriter(recordFile)
	assetInfo := d.exchange.AssetsInfo(pair)

	// Setup progress tracking
	progressBar := progressbar.Default(int64(candleCount))

	// Write CSV headers
	if err := writer.Write(csvHeaders); err != nil {
		return err
	}

	// Download and write candle data
	missingCandles, err := d.downloadCandleBatches(
		ctx,
		pair,
		timeframe,
		parameters.Start,
		parameters.End,
		interval,
		assetInfo.QuotePrecision,
		writer,
		progressBar,
	)
	if err != nil {
		return err
	}

	if err = progressBar.Close(); err != nil {
		log.Warnf("Failed to close progress bar: %s", err.Error())
	}

	if missingCandles > 0 {
		log.Warnf("%d missing candles", missingCandles)
	}

	writer.Flush()
	log.Info("Done!")
	return writer.Error()
}

// initializeParameters creates default parameters for the last month
func initializeParameters() *Parameters {
	now := time.Now()
	return &Parameters{
		Start: now.AddDate(0, -1, 0),
		End:   now,
	}
}

// normalizeTimeParameters adjusts time parameters to appropriate boundaries
func normalizeTimeParameters(parameters *Parameters) {
	// Set start time to beginning of day
	parameters.Start = time.Date(
		parameters.Start.Year(),
		parameters.Start.Month(),
		parameters.Start.Day(),
		0, 0, 0, 0, time.UTC,
	)

	now := time.Now()
	// Ensure end time is not in the future
	if now.Sub(parameters.End) > 0 {
		parameters.End = time.Date(
			parameters.End.Year(),
			parameters.End.Month(),
			parameters.End.Day(),
			0, 0, 0, 0, time.UTC,
		)
	} else {
		parameters.End = now
	}
}

// downloadCandleBatches downloads candles in batches and writes them to CSV
func (d Downloader) downloadCandleBatches(
	ctx context.Context,
	pair string,
	timeframe string,
	start time.Time,
	end time.Time,
	interval time.Duration,
	precision int,
	writer *csv.Writer,
	progressBar *progressbar.ProgressBar,
) (int, error) {
	missingCandles := 0

	for batchStart := start; batchStart.Before(end); batchStart = batchStart.Add(interval * batchSize) {
		batchEnd := calculateBatchEnd(batchStart, interval, end)
		isLastBatch := batchEnd.Equal(end)

		candles, err := d.exchange.CandlesByPeriod(ctx, pair, timeframe, batchStart, batchEnd)
		if err != nil {
			return missingCandles, err
		}

		if err := writeCandles(writer, candles, precision); err != nil {
			return missingCandles, err
		}

		// Update missing candles count
		if !isLastBatch && len(candles) < batchSize {
			missingCandles += batchSize - len(candles)
		}

		// Update progress bar
		if err := progressBar.Add(len(candles)); err != nil {
			log.Warnf("Failed to update progress bar: %s", err.Error())
		}
	}

	return missingCandles, nil
}

// calculateBatchEnd determines the end time for a batch
func calculateBatchEnd(batchStart time.Time, interval time.Duration, totalEnd time.Time) time.Time {
	potentialEnd := batchStart.Add(interval * batchSize)

	// Adjust to ensure we don't go past the final end time
	if potentialEnd.Before(totalEnd) {
		// Subtract 1 second to avoid overlapping with next batch's start
		return potentialEnd.Add(-1 * time.Second)
	}

	return totalEnd
}

// writeCandles writes a batch of candles to the CSV writer
func writeCandles(writer *csv.Writer, candles []core.Candle, precision int) error {
	for _, candle := range candles {
		if err := writer.Write(candle.ToSlice(precision)); err != nil {
			return err
		}
	}
	return nil
}
