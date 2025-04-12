package main

import (
	"fmt"
	"os"
	"time"

	"github.com/raykavin/backnrun"
	"github.com/raykavin/backnrun/internal/backtesting"
	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/exchange/binance"
	"github.com/spf13/cobra"
)

const (
	dateLayout = "2006-01-02"
)

// Command line flags
var (
	// Download command flags
	pair       string
	days       int
	startDate  string
	endDate    string
	timeframe  string
	outputFile string
	isFutures  bool
)

func main() {
	// Create root command
	rootCmd := &cobra.Command{
		Use:     "backnrun",
		Short:   "Utilities for bot automation",
		Version: "1.0.0",
	}

	// Add commands
	rootCmd.AddCommand(buildDownloadCmd())

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func buildDownloadCmd() *cobra.Command {
	downloadCmd := &cobra.Command{
		Use:   "download",
		Short: "Download historical data",
		RunE:  runDownload,
	}

	// Add flags
	downloadCmd.Flags().StringVarP(&pair, "pair", "p", "", "Trading pair (e.g. BTCUSDT)")
	downloadCmd.Flags().IntVarP(&days, "days", "d", 0, "Number of days to download (default 30 days)")
	downloadCmd.Flags().StringVarP(&startDate, "start", "s", "", "Start date (e.g. 2021-12-01)")
	downloadCmd.Flags().StringVarP(&endDate, "end", "e", "", "End date (e.g. 2020-12-31)")
	downloadCmd.Flags().StringVarP(&timeframe, "timeframe", "t", "", "Timeframe (e.g. 1h)")
	downloadCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path (e.g. ./btc.csv)")
	downloadCmd.Flags().BoolVarP(&isFutures, "futures", "f", false, "Use futures market")

	// Required flags
	downloadCmd.MarkFlagRequired("pair")
	downloadCmd.MarkFlagRequired("timeframe")
	downloadCmd.MarkFlagRequired("output")

	return downloadCmd
}

func runDownload(cmd *cobra.Command, args []string) error {
	// Initialize exchange
	exc, err := initializeExchange(cmd)
	if err != nil {
		return err
	}

	// Build download options
	options, err := buildDownloadOptions()
	if err != nil {
		return err
	}

	// Run the download
	return backtesting.NewDownloader(exc, backnrun.DefaultLog).Download(
		cmd.Context(),
		pair,
		timeframe,
		outputFile,
		options...,
	)
}

func initializeExchange(cmd *cobra.Command) (core.Feeder, error) {
	exchangeType := binance.MarketTypeSpot
	if isFutures {
		exchangeType = binance.MarketTypeFutures
	}

	return binance.NewExchange(cmd.Context(), backnrun.DefaultLog, binance.Config{
		Type: exchangeType,
	})

}

func buildDownloadOptions() ([]backtesting.Option, error) {
	var options []backtesting.Option

	// Add days option if specified
	if days > 0 {
		options = append(options, backtesting.WithDays(days))
	}

	// Handle date range options
	if startDate != "" || endDate != "" {
		// Both must be provided together
		if startDate == "" || endDate == "" {
			return nil, fmt.Errorf("START and END dates must be provided together")
		}

		// Parse dates
		start, err := time.Parse(dateLayout, startDate)
		if err != nil {
			return nil, fmt.Errorf("invalid start date format: %w", err)
		}

		end, err := time.Parse(dateLayout, endDate)
		if err != nil {
			return nil, fmt.Errorf("invalid end date format: %w", err)
		}

		options = append(options, backtesting.WithInterval(start, end))
	}

	return options, nil
}
