# BackNRun

BackNRun is a powerful, flexible trading bot framework written in Go. It provides a comprehensive set of tools for developing, backtesting, and optimizing trading strategies for cryptocurrency markets.


## 📌 Roadmap & Features

- [x] **Backtesting**: Test strategies against historical data to evaluate performance 
- [x] **Strategy Development**: Create custom trading strategies using a simple, extensible interface
- [x] **Notifications** - Telegram Notifications: Implemented notifications from Telegram channel
- [x] **Parameter Optimization**: Find optimal parameters for your strategies using grid search or random search  
- [x] **Performance Metrics**: Analyze strategy performance with comprehensive metrics  
- [ ] **Web dashboard for live tracking:** Plot trading results with indicators and trades  

---
## 📁 Architecture

BackNRun is built with a modular architecture that separates concerns and allows for easy extension:

```
📁 backnrun/
├── 📁 cmd/                  # Command-line application
├── 📁 examples/             # Example strategies and usage
├── 📁 internal/             # Internal packages
├── 📁 pkg/                  # Core packages
│   ├──📁 core/             # Core interfaces and types
│   ├──📁 exchange/         # Exchange implementations
│   ├──📁 indicator/        # Technical indicators
│   ├──📁 logger/           # Logging utilities
│   ├──📁 metric/           # Performance metrics
│   ├──📁 notification/     # Notification systems
│   ├──📁 optimizer/        # Strategy parameter optimization
│   ├──📁 order/            # Order management
│   ├──📁 plot/             # Visualization tools
│   ├──📁 storage/          # Data storage
│   └──📁 strategy/         # Strategy implementations
```

## 📦 Installation

### Prerequisites

- Go 1.23 or higher
- Git

### Building from Source

```bash
# Clone the repository
git clone https://github.com/raykavin/backnrun.git
cd backnrun

# Build the application
go build -o backnrun cmd/backnrun/main.go
```

## ⚡ Quick Start

### Downloading Historical Data

```bash
# Download 30 days of BTC/USDT 15-minute candles
./backnrun download -p BTCUSDT -t 15m -d 30 -o btc-15m.csv
```

### Running a Backtest

Create a Go file with your backtest configuration:

```go
package main

import (
	"context"

	"github.com/raykavin/backnrun"
	"github.com/raykavin/backnrun/examples/strategies"
	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/exchange"
	"github.com/raykavin/backnrun/pkg/logger"
	"github.com/raykavin/backnrun/pkg/plot"
	"github.com/raykavin/backnrun/pkg/storage"
)

func main() {
	// Set up context and logging
	ctx := context.Background()
	log := backnrun.DefaultLog
	log.SetLevel(logger.DebugLevel)

	// Initialize trading strategy
	strategy := strategies.NewCrossEMA(9, 21, 10.0)

	// Configure trading pairs
	settings := &core.Settings{
		Pairs: []string{"BTCUSDT"},
	}

	// Initialize data feed
	dataFeed, err := exchange.NewCSVFeed(
		strategy.Timeframe(),
		exchange.PairFeed{
			Pair:      "BTCUSDT",
			File:      "btc-15m.csv",
			Timeframe: "15m",
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize in-memory storage
	db, err := storage.FromMemory()
	if err != nil {
		log.Fatal(err)
	}

	// Initialize paper wallet
	wallet := exchange.NewPaperWallet(
		ctx,
		"USDT",
		log,
		exchange.WithPaperAsset("USDT", 1000),
		exchange.WithDataFeed(dataFeed),
	)

	// Initialize chart
	chart, err := plot.NewChart(
		log,
		plot.WithStrategyIndicators(strategy),
		plot.WithPaperWallet(wallet),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Set up the trading bot
	bot, err := backnrun.NewBot(
		ctx,
		settings,
		wallet,
		strategy,
		log,
		backnrun.WithBacktest(wallet),
		backnrun.WithStorage(db),
		backnrun.WithCandleSubscription(chart),
		backnrun.WithOrderSubscription(chart),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Run simulation
	if err := bot.Run(ctx); err != nil {
		log.Fatal(err)
	}

	// Display results
	bot.Summary()

	// Show interactive chart
	if err := chart.Start(); err != nil {
		log.Fatal(err)
	}
}
```

Run the backtest:

```bash
go run backtest.go
```

## 🤖 Available Strategies

BackNRun comes with several example strategies:

- **EMA Cross**: Trading based on exponential moving average crossovers
- **MACD Divergence**: Trading based on MACD indicator divergence
- **Triple EMA Cross**: Trading using three exponential moving averages
- **Trend Master**: Trend-following strategy with multiple indicators
- **Turtle Trading**: Implementation of the famous Turtle Trading system
- **Larry Williams 91**: Based on Larry Williams' trading methodology
- **Trailing Stop**: Strategy with dynamic trailing stop-loss
- **OCO Sell**: One-Cancels-the-Other order strategy

## Creating Custom Strategies

To create a custom strategy, implement the `core.Strategy` interface:

```go
type Strategy interface {
	// Timeframe is the time interval in which the strategy will be executed. eg: 1h, 1d, 1w
	Timeframe() string
	// WarmupPeriod is the necessary time to wait before executing the strategy, to load data for indicators.
	// This time is measured in the period specified in the `Timeframe` function.
	WarmupPeriod() int
	// Indicators will be executed for each new candle, in order to fill indicators before `OnCandle` function is called.
	Indicators(df *Dataframe) []ChartIndicator
	// OnCandle will be executed for each new candle, after indicators are filled, here you can do your trading logic.
	// OnCandle is executed after the candle close.
	OnCandle(df *Dataframe, broker Broker)
}
```

Example of a simple strategy:

```go
type MyStrategy struct {
	// Strategy parameters
	fastPeriod int
	slowPeriod int
}

func NewMyStrategy() *MyStrategy {
	return &MyStrategy{
		fastPeriod: 9,
		slowPeriod: 21,
	}
}

func (s *MyStrategy) Timeframe() string {
	return "1h"
}

func (s *MyStrategy) WarmupPeriod() int {
	return 100
}

func (s *MyStrategy) Indicators(df *core.Dataframe) []core.ChartIndicator {
	// Calculate indicators
	df.Metadata["fast"] = indicator.EMA(df.Close, s.fastPeriod)
	df.Metadata["slow"] = indicator.EMA(df.Close, s.slowPeriod)

	// Return chart indicators for visualization
	return []core.ChartIndicator{
		{
			Overlay: true,
			GroupName: "Moving Averages",
			Time: df.Time,
			Metrics: []core.IndicatorMetric{
				{
					Values: df.Metadata["fast"],
					Name:   "Fast EMA",
					Color:  "red",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["slow"],
					Name:   "Slow EMA",
					Color:  "blue",
					Style:  core.StyleLine,
				},
			},
		},
	}
}

func (s *MyStrategy) OnCandle(df *core.Dataframe, broker core.Broker) {
	// Get current position
	assetPosition, quotePosition, err := broker.Position(df.Pair)
	if err != nil {
		return
	}

	// Get indicator values
	fast := df.Metadata["fast"].Last(0)
	slow := df.Metadata["slow"].Last(0)
	
	// Buy signal: fast EMA crosses above slow EMA
	if fast > slow && df.Metadata["fast"].Last(1) <= df.Metadata["slow"].Last(1) && quotePosition > 0 {
		// Calculate position size
		price := df.Close.Last(0)
		amount := quotePosition / price
		
		// Create market buy order
		broker.CreateOrderMarket(core.SideTypeBuy, df.Pair, amount)
	}
	
	// Sell signal: fast EMA crosses below slow EMA
	if fast < slow && df.Metadata["fast"].Last(1) >= df.Metadata["slow"].Last(1) && assetPosition > 0 {
		// Create market sell order
		broker.CreateOrderMarket(core.SideTypeSell, df.Pair, assetPosition)
	}
}
```

## 📊 Parameter Optimization

BackNRun includes a powerful parameter optimization package that helps you find the best parameters for your trading strategies. The optimizer supports multiple algorithms and performance metrics.

### Optimization Algorithms

- **Grid Search**: Exhaustively tests all combinations of parameter values within specified ranges
- **Random Search**: Tests random combinations of parameter values, which can be more efficient for high-dimensional parameter spaces

### Performance Metrics

The optimizer tracks various performance metrics:

- `profit`: Total profit
- `win_rate`: Percentage of winning trades
- `payoff`: Payoff ratio (average win / average loss)
- `profit_factor`: Profit factor (gross profit / gross loss)
- `sqn`: System Quality Number
- `drawdown`: Maximum drawdown
- `sharpe_ratio`: Sharpe ratio
- `trade_count`: Total number of trades

### Example: Optimizing a Strategy

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/raykavin/backnrun"
	"github.com/raykavin/backnrun/examples/strategies"
	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/exchange"
	"github.com/raykavin/backnrun/pkg/logger"
	"github.com/raykavin/backnrun/pkg/optimizer"
)

func main() {
	// Set up context and logging
	ctx := context.Background()
	log := backnrun.DefaultLog
	log.SetLevel(logger.InfoLevel)

	// Initialize data feed for backtesting
	dataFeed, err := exchange.NewCSVFeed(
		"15m",
		exchange.PairFeed{
			Pair:      "BTCUSDT",
			File:      "btc-15m.csv",
			Timeframe: "15m",
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	// Configure trading pairs
	settings := &core.Settings{
		Pairs: []string{"BTCUSDT"},
	}

	// Create strategy factory for EMA Cross strategy
	strategyFactory := func(params optimizer.ParameterSet) (core.Strategy, error) {
		// Extract parameters with validation
		emaLength, ok := params["emaLength"].(int)
		if !ok {
			return nil, fmt.Errorf("emaLength must be an integer")
		}

		smaLength, ok := params["smaLength"].(int)
		if !ok {
			return nil, fmt.Errorf("smaLength must be an integer")
		}

		minQuoteAmount, ok := params["minQuoteAmount"].(float64)
		if !ok {
			return nil, fmt.Errorf("minQuoteAmount must be a float")
		}

		// Create and return the strategy
		return strategies.NewCrossEMA(emaLength, smaLength, minQuoteAmount), nil
	}

	// Create evaluator
	evaluator := optimizer.NewBacktestStrategyEvaluator(
		strategyFactory,
		settings,
		dataFeed,
		log,
		1000.0, // Starting balance
		"USDT", // Quote currency
	)

	// Define parameters for optimization
	parameters := []optimizer.Parameter{
		{
			Name:        "emaLength",
			Description: "Length of the EMA indicator",
			Default:     9,
			Min:         3,
			Max:         50,
			Step:        1,
			Type:        optimizer.TypeInt,
		},
		{
			Name:        "smaLength",
			Description: "Length of the SMA indicator",
			Default:     21,
			Min:         5,
			Max:         100,
			Step:        5,
			Type:        optimizer.TypeInt,
		},
		{
			Name:        "minQuoteAmount",
			Description: "Minimum quote currency amount for trades",
			Default:     10.0,
			Min:         1.0,
			Max:         100.0,
			Step:        5.0,
			Type:        optimizer.TypeFloat,
		},
	}

	// Configure optimizer
	config := optimizer.NewConfig().
		WithParameters(parameters...).
		WithMaxIterations(50).
		WithParallelism(4).
		WithLogger(log).
		WithTargetMetric(optimizer.MetricProfit, true)

	// Create grid search optimizer
	gridSearch, err := optimizer.NewGridSearch(config)
	if err != nil {
		log.Fatal(err)
	}

	// Run optimization
	fmt.Println("Starting grid search optimization...")
	startTime := time.Now()

	results, err := gridSearch.Optimize(
		ctx,
		evaluator,
		optimizer.MetricProfit,
		true,
	)
	if err != nil {
		log.Fatal(err)
	}

	duration := time.Since(startTime)
	fmt.Printf("Optimization completed in %s\n", duration.Round(time.Second))

	// Print results
	optimizer.PrintResults(results, optimizer.MetricProfit, 5)

	// Save results to CSV
	outputFile := "ema_optimization_results.csv"
	if err := optimizer.SaveResultsToCSV(results, optimizer.MetricProfit, outputFile); err != nil {
		log.Errorf("Failed to save results: %v", err)
	} else {
		fmt.Printf("Results saved to %s\n", outputFile)
	}
}
```

## 🛑 Live Trading

To use BackNRun for live trading, you need to configure it with a real exchange:

```go
package main

import (
	"context"

	"github.com/raykavin/backnrun"
	"github.com/raykavin/backnrun/examples/strategies"
	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/exchange/binance"
	"github.com/raykavin/backnrun/pkg/logger"
	"github.com/raykavin/backnrun/pkg/notification"
)

func main() {
	// Set up context and logging
	ctx := context.Background()
	log := backnrun.DefaultLog
	log.SetLevel(logger.InfoLevel)

	// Initialize trading strategy
	strategy := strategies.NewCrossEMA(9, 21, 10.0)

	// Configure trading pairs
	settings := &core.Settings{
		Pairs: []string{"BTCUSDT"},
	}

	// Initialize Binance exchange
	exchange, err := binance.NewExchange(ctx, log, binance.Config{
		Type:      binance.MarketTypeSpot,
		ApiKey:    "YOUR_API_KEY",
		ApiSecret: "YOUR_API_SECRET",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Initialize Telegram notifications (optional)
	telegramNotifier, err := notification.NewTelegram(
		"YOUR_TELEGRAM_BOT_TOKEN",
		"YOUR_TELEGRAM_CHAT_ID",
		log,
	)
	if err != nil {
		log.Fatal(err)
	}

	// Set up the trading bot
	bot, err := backnrun.NewBot(
		ctx,
		settings,
		exchange,
		strategy,
		log,
		backnrun.WithTelegram(telegramNotifier),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Run the bot
	if err := bot.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
```

## 🤝 Contributing

Contributions to BackNRun are welcome! Here are some ways you can contribute:

1. Report bugs and suggest features by opening issues
2. Submit pull requests with bug fixes or new features
3. Improve documentation
4. Share your custom strategies with the community

## 📄License

MIT License © [Raykavin Meireles](https://github.com/raykavin)

BackNRun is licensed under the MIT License. See the [LICENSE](LICENSE.md) file for details.

---
## 📬 Contact

Feel free to reach out for support or collaboration:  
**Email**: [raykavin.meireles@gmail.com](mailto:raykavin.meireles@gmail.com)  
**GitHub**: [@raykavin](https://github.com/raykavin)\
**LinkedIn**: [@raykavin.dev](https://www.linkedin.com/in/raykavin-dev)\
**Instagram**: [@raykavin.dev](https://www.linkedin.com/in/raykavin-dev)