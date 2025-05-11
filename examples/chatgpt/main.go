package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/raykavin/backnrun/bot"
	"github.com/raykavin/backnrun/core"
	"github.com/raykavin/backnrun/exchange"
	"github.com/raykavin/backnrun/exchange/binance"
	"github.com/raykavin/backnrun/plot"
	"github.com/raykavin/backnrun/storage"
	"github.com/raykavin/backnrun/strategies"
)

func main() {
	// Parse command line flags
	apiKey := flag.String("api-key", "", "OpenAI API key")
	symbol := flag.String("symbol", "BTCUSDT", "Trading symbol")
	useBacktest := flag.Bool("backtest", false, "Use backtesting mode")
	startDate := flag.String("start-date", "", "Start date for backtesting (format: 2006-01-02)")
	endDate := flag.String("end-date", "", "End date for backtesting (format: 2006-01-02)")
	initialBalance := flag.Float64("balance", 10000.0, "Initial balance for paper wallet")
	model := flag.String("model", "gpt-4-turbo-preview", "OpenAI model to use")
	analysisInterval := flag.Int("interval", 12, "Number of candles between analyses")
	enableChart := flag.Bool("chart", true, "Enable chart visualization")
	dataDir := flag.String("data-dir", "./data", "Directory for CSV data files (for CSV backtest)")
	useCsv := flag.Bool("use-csv", false, "Use CSV files for backtesting instead of Binance API")

	flag.Parse()

	godotenv.Load()

	// Check if API key is provided
	if *apiKey == "" {
		// Try to get API key from environment variable
		*apiKey = os.Getenv("OPENAI_API_KEY")
		if *apiKey == "" {
			log.Fatal("OpenAI API key is required. Provide it with -api-key flag or OPENAI_API_KEY environment variable")
		}
	}

	// Create the ChatGPT strategy with configuration
	strategy := strategies.NewChatGPTStrategy(*apiKey)
	strategy.WithModel(*model).WithAnalysisInterval(*analysisInterval)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	setupSignalHandling(cancel)

	// Set log level
	bot.DefaultLog.SetLevel(core.InfoLevel)

	if *useBacktest {
		if *useCsv {
			runCsvBacktest(ctx, strategy, *symbol, *dataDir, *initialBalance, *enableChart)
		} else {
			runBacktest(ctx, strategy, *symbol, *startDate, *endDate, *initialBalance, *enableChart)
		}
	} else {
		runLive(ctx, strategy, *symbol, *initialBalance, *enableChart)
	}
}

func runLive(ctx context.Context, strategy *strategies.ChatGPTStrategy, symbol string, initialBalance float64, enableChart bool) {
	fmt.Println("Starting ChatGPT trading strategy in live mode with paper wallet")
	fmt.Println("Trading symbol:", symbol)
	fmt.Println("Initial balance:", initialBalance)

	// Create settings
	settings := &core.Settings{
		Pairs: []string{symbol},
	}

	// Create exchange for market data
	binanceClient, err := binance.NewExchange(ctx, bot.DefaultLog, binance.Config{
		Type: binance.MarketTypeSpot,
	})
	if err != nil {
		log.Fatalf("Failed to create exchange: %v", err)
	}

	// Create paper wallet for simulated trading
	paperWallet := exchange.NewPaperWallet(
		ctx,
		"USDT",
		bot.DefaultLog,
		exchange.WithPaperFee(0.001, 0.001),
		exchange.WithPaperAsset("USDT", initialBalance),
		exchange.WithDataFeed(binanceClient),
	)

	// Create in-memory storage
	memStorage, err := storage.FromMemory()
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}

	// Create bot options
	botOptions := []bot.Option{
		bot.WithStorage(memStorage),
		bot.WithPaperWallet(paperWallet),
	}

	// Setup chart visualization if enabled
	var chartServer *plot.ChartServer
	var chart *plot.Chart
	if enableChart {
		chart, chartServer = setupChart(strategy, paperWallet)
		botOptions = append(botOptions,
			bot.WithCandleSubscription(chart),
			bot.WithOrderSubscription(chart),
		)
	}

	// Create and start the trading bot
	b, err := bot.NewBot(
		ctx,
		settings,
		paperWallet,
		bot.DefaultLog,
		strategy,
		botOptions...,
	)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Start the chart server if enabled
	if enableChart && chartServer != nil {
		go func() {
			if err := chartServer.Start(); err != nil {
				log.Printf("Chart server error: %v", err)
			}
		}()
		fmt.Println("Chart server started. Open http://localhost:8080 in your browser to view the chart.")
	}

	// Start the bot
	if err := b.Run(ctx); err != nil {
		log.Fatalf("Failed to run bot: %v", err)
	}

	// Wait for context cancellation
	<-ctx.Done()
	fmt.Println("Shutting down...")
}

func runBacktest(ctx context.Context, strategy *strategies.ChatGPTStrategy, symbol, startDateStr, endDateStr string, initialBalance float64, enableChart bool) {
	fmt.Println("Starting ChatGPT trading strategy in backtest mode")
	fmt.Println("Trading symbol:", symbol)
	fmt.Println("Initial balance:", initialBalance)

	// Parse dates
	var startDate, endDate time.Time
	var err error

	if startDateStr != "" {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			log.Fatalf("Invalid start date format: %v", err)
		}
	} else {
		// Default to 30 days ago
		startDate = time.Now().AddDate(0, 0, -30)
	}

	if endDateStr != "" {
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			log.Fatalf("Invalid end date format: %v", err)
		}
	} else {
		// Default to now
		endDate = time.Now()
	}

	fmt.Printf("Backtest period: %s to %s\n", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	// Create settings for backtest
	settings := &core.Settings{
		Pairs: []string{symbol},
	}

	// Create exchange for historical data
	binanceClient, err := binance.NewExchange(ctx, bot.DefaultLog, binance.Config{
		Type: binance.MarketTypeSpot,
	})
	if err != nil {
		log.Fatalf("Failed to create exchange: %v", err)
	}

	// Create paper wallet for simulated trading
	paperWallet := exchange.NewPaperWallet(
		ctx,
		"USDT",
		bot.DefaultLog,
		exchange.WithPaperFee(0.001, 0.001),
		exchange.WithPaperAsset("USDT", initialBalance),
		exchange.WithDataFeed(binanceClient),
	)

	// Create in-memory storage
	memStorage, err := storage.FromMemory()
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}

	// Create bot options
	botOptions := []bot.Option{
		bot.WithStorage(memStorage),
		bot.WithPaperWallet(paperWallet),
		bot.WithBacktest(paperWallet),
	}

	// Setup chart visualization if enabled
	var chartServer *plot.ChartServer
	var chart *plot.Chart
	if enableChart {
		chart, chartServer = setupChart(strategy, paperWallet)
		botOptions = append(botOptions,
			bot.WithCandleSubscription(chart),
			bot.WithOrderSubscription(chart),
		)
	}

	// Create and start the trading bot in backtest mode
	b, err := bot.NewBot(
		ctx,
		settings,
		paperWallet,
		bot.DefaultLog,
		strategy,
		botOptions...,
	)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Start the chart server if enabled
	if enableChart && chartServer != nil {
		go func() {
			if err := chartServer.Start(); err != nil {
				log.Printf("Chart server error: %v", err)
			}
		}()
		fmt.Println("Chart server started. Open http://localhost:8080 in your browser to view the chart.")
	}

	// Start the bot
	if err := b.Run(ctx); err != nil {
		log.Fatalf("Failed to run bot: %v", err)
	}

	// Wait for backtest to complete
	<-ctx.Done()

	// Print backtest results
	printBacktestResults(ctx, paperWallet, initialBalance)
}

func runCsvBacktest(ctx context.Context, strategy *strategies.ChatGPTStrategy, symbol, dataDir string, initialBalance float64, enableChart bool) {
	fmt.Println("Starting ChatGPT trading strategy in CSV backtest mode")
	fmt.Println("Trading symbol:", symbol)
	fmt.Println("Initial balance:", initialBalance)
	fmt.Println("Data directory:", dataDir)

	// Ensure data directory exists
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			log.Fatalf("Failed to create data directory: %v", err)
		}
	}

	// Check if CSV file exists
	csvFile := filepath.Join(dataDir, fmt.Sprintf("%s-%s.csv", symbol, strategy.Timeframe()))
	if _, err := os.Stat(csvFile); os.IsNotExist(err) {
		log.Fatalf("CSV file not found: %s. Please download historical data first.", csvFile)
	}

	// Create CSV feed
	pairFeed := exchange.PairFeed{
		Pair:      symbol,
		File:      csvFile,
		Timeframe: strategy.Timeframe(),
	}

	csvFeed, err := exchange.NewCSVFeed(strategy.Timeframe(), pairFeed)
	if err != nil {
		log.Fatalf("Failed to create CSV feed: %v", err)
	}

	// Create settings
	settings := &core.Settings{
		Pairs: []string{symbol},
	}

	// Create paper wallet for simulated trading
	paperWallet := exchange.NewPaperWallet(
		ctx,
		"USDT",
		bot.DefaultLog,
		exchange.WithPaperFee(0.001, 0.001),
		exchange.WithPaperAsset("USDT", initialBalance),
		exchange.WithDataFeed(csvFeed),
	)

	// Create in-memory storage
	memStorage, err := storage.FromMemory()
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}

	// Create bot options
	botOptions := []bot.Option{
		bot.WithStorage(memStorage),
		bot.WithPaperWallet(paperWallet),
		bot.WithBacktest(paperWallet),
	}

	// Setup chart visualization if enabled
	var chartServer *plot.ChartServer
	var chart *plot.Chart
	if enableChart {
		chart, chartServer = setupChart(strategy, paperWallet)
		botOptions = append(botOptions,
			bot.WithCandleSubscription(chart),
			bot.WithOrderSubscription(chart),
		)
	}

	// Create and start the trading bot in backtest mode
	b, err := bot.NewBot(
		ctx,
		settings,
		paperWallet,
		bot.DefaultLog,
		strategy,
		botOptions...,
	)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Start the chart server if enabled
	if enableChart && chartServer != nil {
		go func() {
			if err := chartServer.Start(); err != nil {
				log.Printf("Chart server error: %v", err)
			}
		}()
		fmt.Println("Chart server started. Open http://localhost:8080 in your browser to view the chart.")
	}

	// Start the bot
	if err := b.Run(ctx); err != nil {
		log.Fatalf("Failed to run bot: %v", err)
	}

	// Wait for backtest to complete
	<-ctx.Done()

	// Print backtest results
	printBacktestResults(ctx, paperWallet, initialBalance)
}

func setupChart(strategy core.Strategy, wallet *exchange.PaperWallet) (*plot.Chart, *plot.ChartServer) {
	chart, err := plot.NewChart(
		bot.DefaultLog,
		plot.WithStrategyIndicators(strategy),
		plot.WithPaperWallet(wallet),
	)
	if err != nil {
		log.Printf("Failed to create chart: %v", err)
		return nil, nil
	}

	server := plot.NewChartServer(chart, plot.NewStandardHTTPServer(), bot.DefaultLog)
	return chart, server
}

func printBacktestResults(ctx context.Context, paperWallet *exchange.PaperWallet, initialBalance float64) {
	fmt.Println("\nBacktest Results:")
	account, err := paperWallet.Account(ctx)
	if err != nil {
		log.Printf("Failed to get account: %v", err)
		return
	}

	usdtBalance, _ := account.GetBalance("USDT", "")

	balance := usdtBalance.Free + usdtBalance.Lock
	profitPct := (balance - initialBalance) / initialBalance * 100

	fmt.Printf("Initial Balance: %.2f USDT\n", initialBalance)
	fmt.Printf("Final Balance: %.2f USDT\n", balance)
	fmt.Printf("Profit/Loss: %.2f USDT (%.2f%%)\n", balance-initialBalance, profitPct)

	// Note: Trade summary is not available in this version
	// In a production system, you would implement trade tracking
	fmt.Println("\nNote: Detailed trade statistics are not available in this example.")
	fmt.Println("To track trades, you would need to implement a custom trade tracking mechanism.")
}

func setupSignalHandling(cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("\nReceived shutdown signal")
		cancel()
	}()
}
