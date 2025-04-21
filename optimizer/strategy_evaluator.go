package optimizer

import (
	"context"
	"fmt"
	"time"

	"github.com/raykavin/backnrun/bot"
	"github.com/raykavin/backnrun/core"
	"github.com/raykavin/backnrun/exchange"
	"github.com/raykavin/backnrun/storage"
	"github.com/raykavin/backnrun/strategies"
)

// StrategyFactory is a function that creates a strategy with the given parameters
type StrategyFactory func(params core.ParameterSet) (core.Strategy, error)

// BacktestStrategyEvaluator evaluates a strategy using backtesting
type BacktestStrategyEvaluator struct {
	strategyFactory StrategyFactory
	settings        *core.Settings
	dataFeed        *exchange.CSVFeed
	logger          core.Logger
	startBalance    float64
	quoteCurrency   string
}

// NewBacktestStrategyEvaluator creates a new evaluator for backtesting strategies
func NewBacktestStrategyEvaluator(
	strategyFactory StrategyFactory,
	settings *core.Settings,
	dataFeed *exchange.CSVFeed,
	logger core.Logger,
	startBalance float64,
	quoteCurrency string,
) *BacktestStrategyEvaluator {
	return &BacktestStrategyEvaluator{
		strategyFactory: strategyFactory,
		settings:        settings,
		dataFeed:        dataFeed,
		logger:          logger,
		startBalance:    startBalance,
		quoteCurrency:   quoteCurrency,
	}
}

// Evaluate runs a backtest with the given parameters and returns performance metrics
func (e *BacktestStrategyEvaluator) Evaluate(ctx context.Context, params core.ParameterSet) (*core.OptimizerResult, error) {
	startTime := time.Now()

	// Create strategy with the given parameters
	strategy, err := e.strategyFactory(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create strategy: %w", err)
	}

	// Initialize in-memory storage
	db, err := storage.FromMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Initialize paper wallet with starting balance
	wallet := exchange.NewPaperWallet(
		ctx,
		e.quoteCurrency,
		e.logger,
		exchange.WithPaperAsset(e.quoteCurrency, e.startBalance),
		exchange.WithDataFeed(e.dataFeed),
	)

	// Set up the trading bot
	bot, err := bot.NewBot(
		ctx,
		e.settings,
		wallet,
		strategy,
		e.logger,
		bot.WithBacktest(wallet),
		bot.WithStorage(db),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize bot: %w", err)
	}

	// Run the backtest
	if err := bot.Run(ctx); err != nil {
		return nil, fmt.Errorf("backtest failed: %w", err)
	}

	// Collect metrics
	metrics, err := e.collectMetrics(bot)
	if err != nil {
		return nil, fmt.Errorf("failed to collect metrics: %w", err)
	}

	// Create result
	result := &core.OptimizerResult{
		Parameters: params,
		Metrics:    metrics,
		Duration:   time.Since(startTime),
	}

	return result, nil
}

// collectMetrics extracts performance metrics from the bot after a backtest
func (e *BacktestStrategyEvaluator) collectMetrics(bot *bot.Bot) (map[string]float64, error) {
	metrics := make(map[string]float64)

	// Initialize counters
	totalProfit := 0.0
	totalWins := 0
	totalLosses := 0
	totalTrades := 0
	avgPayoff := 0.0
	avgProfitFactor := 0.0
	avgSQN := 0.0

	// Process results for each pair
	for _, summary := range bot.Controller().Results {
		wins := len(summary.Win())
		losses := len(summary.Lose())
		trades := wins + losses

		if trades > 0 {
			totalProfit += summary.Profit()
			totalWins += wins
			totalLosses += losses
			totalTrades += trades

			// Calculate metrics
			winRate := float64(wins) / float64(trades)
			payoff := summary.Payoff()
			profitFactor := summary.ProfitFactor()
			sqn := summary.SQN()

			// Weight metrics by number of trades
			avgPayoff += payoff * float64(trades)
			avgProfitFactor += profitFactor * float64(trades)
			avgSQN += sqn

			// Store pair-specific metrics
			metrics[fmt.Sprintf("%s_profit", summary.Pair)] = summary.Profit()
			metrics[fmt.Sprintf("%s_win_rate", summary.Pair)] = winRate
			metrics[fmt.Sprintf("%s_payoff", summary.Pair)] = payoff
			metrics[fmt.Sprintf("%s_profit_factor", summary.Pair)] = profitFactor
			metrics[fmt.Sprintf("%s_sqn", summary.Pair)] = sqn
			metrics[fmt.Sprintf("%s_trades", summary.Pair)] = float64(trades)
		}
	}

	// Calculate overall metrics
	if totalTrades > 0 {
		metrics[string(core.MetricProfit)] = totalProfit
		metrics[string(core.MetricWinRate)] = float64(totalWins) / float64(totalTrades)
		metrics[string(core.MetricPayoff)] = avgPayoff / float64(totalTrades)
		metrics[string(core.MetricProfitFactor)] = avgProfitFactor / float64(totalTrades)
		metrics[string(core.MetricTradeCount)] = float64(totalTrades)

		// Average SQN across pairs
		if len(bot.Controller().Results) > 0 {
			metrics[string(core.MetricSQN)] = avgSQN / float64(len(bot.Controller().Results))
		}
	} else {
		// No trades executed
		metrics[string(core.MetricProfit)] = 0
		metrics[string(core.MetricWinRate)] = 0
		metrics[string(core.MetricPayoff)] = 0
		metrics[string(core.MetricProfitFactor)] = 0
		metrics[string(core.MetricSQN)] = 0
		metrics[string(core.MetricTradeCount)] = 0
	}

	// We can't directly access the paper wallet as it's unexported
	// The metrics we need are already calculated in the summary
	// Just ensure we have profit metrics
	if totalProfit != 0 {
		metrics["final_balance"] = e.startBalance + totalProfit
		metrics["return_pct"] = (totalProfit / e.startBalance) * 100
	} else {
		metrics["final_balance"] = e.startBalance
		metrics["return_pct"] = 0
	}

	return metrics, nil
}

// CreateEMAStrategyFactory creates a factory function for EMA cross strategies
// This uses the strategies package from examples
func CreateEMAStrategyFactory() StrategyFactory {
	return func(params core.ParameterSet) (core.Strategy, error) {
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
}
