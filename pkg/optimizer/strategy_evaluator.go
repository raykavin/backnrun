package optimizer

import (
	"context"
	"fmt"
	"time"

	"github.com/raykavin/backnrun"
	"github.com/raykavin/backnrun/examples/strategies"
	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/exchange"
	"github.com/raykavin/backnrun/pkg/logger"
	"github.com/raykavin/backnrun/pkg/storage"
)

// StrategyFactory is a function that creates a strategy with the given parameters
type StrategyFactory func(params ParameterSet) (core.Strategy, error)

// BacktestStrategyEvaluator evaluates a strategy using backtesting
type BacktestStrategyEvaluator struct {
	strategyFactory StrategyFactory
	settings        *core.Settings
	dataFeed        *exchange.CSVFeed
	logger          logger.Logger
	startBalance    float64
	quoteCurrency   string
}

// NewBacktestStrategyEvaluator creates a new evaluator for backtesting strategies
func NewBacktestStrategyEvaluator(
	strategyFactory StrategyFactory,
	settings *core.Settings,
	dataFeed *exchange.CSVFeed,
	logger logger.Logger,
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
func (e *BacktestStrategyEvaluator) Evaluate(ctx context.Context, params ParameterSet) (*Result, error) {
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
	bot, err := backnrun.NewBot(
		ctx,
		e.settings,
		wallet,
		strategy,
		e.logger,
		backnrun.WithBacktest(wallet),
		backnrun.WithStorage(db),
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
	result := &Result{
		Parameters: params,
		Metrics:    metrics,
		Duration:   time.Since(startTime),
	}

	return result, nil
}

// collectMetrics extracts performance metrics from the bot after a backtest
func (e *BacktestStrategyEvaluator) collectMetrics(bot *backnrun.Bot) (map[string]float64, error) {
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
		metrics[string(MetricProfit)] = totalProfit
		metrics[string(MetricWinRate)] = float64(totalWins) / float64(totalTrades)
		metrics[string(MetricPayoff)] = avgPayoff / float64(totalTrades)
		metrics[string(MetricProfitFactor)] = avgProfitFactor / float64(totalTrades)
		metrics[string(MetricTradeCount)] = float64(totalTrades)
		
		// Average SQN across pairs
		if len(bot.Controller().Results) > 0 {
			metrics[string(MetricSQN)] = avgSQN / float64(len(bot.Controller().Results))
		}
	} else {
		// No trades executed
		metrics[string(MetricProfit)] = 0
		metrics[string(MetricWinRate)] = 0
		metrics[string(MetricPayoff)] = 0
		metrics[string(MetricProfitFactor)] = 0
		metrics[string(MetricSQN)] = 0
		metrics[string(MetricTradeCount)] = 0
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

// OptimizableStrategy is an interface for strategies that can be optimized
type OptimizableStrategy interface {
	core.Strategy
	StrategyEvaluator
}

// CreateEMAStrategyFactory creates a factory function for EMA cross strategies
// This uses the strategies package from examples
func CreateEMAStrategyFactory() StrategyFactory {
	return func(params ParameterSet) (core.Strategy, error) {
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

// CreateMACDStrategyFactory creates a factory function for MACD strategies
// This uses the strategies package from examples
func CreateMACDStrategyFactory() StrategyFactory {
	return func(params ParameterSet) (core.Strategy, error) {
		// Create a new MACD strategy with default parameters
		strategy := strategies.NewMACDDivergenceStrategy()
		
		// Since the strategy methods for setting parameters might not be exported,
		// we would need to modify the strategy implementation to support parameter changes
		// For now, we'll return the strategy with default parameters
		
		// Note: In a real implementation, you would need to ensure the strategy
		// has proper setters for these parameters or modify the strategy to accept
		// parameters in its constructor
		
		return strategy, nil
	}
}

// GetEMAStrategyParameters returns the parameters that can be optimized for EMA cross strategy
func GetEMAStrategyParameters() []Parameter {
	return []Parameter{
		{
			Name:        "emaLength",
			Description: "Length of the EMA indicator",
			Default:     9,
			Min:         3,
			Max:         50,
			Step:        1,
			Type:        TypeInt,
		},
		{
			Name:        "smaLength",
			Description: "Length of the SMA indicator",
			Default:     21,
			Min:         5,
			Max:         100,
			Step:        1,
			Type:        TypeInt,
		},
		{
			Name:        "minQuoteAmount",
			Description: "Minimum quote currency amount for trades",
			Default:     10.0,
			Min:         1.0,
			Max:         100.0,
			Step:        5.0,
			Type:        TypeFloat,
		},
	}
}

// GetMACDStrategyParameters returns the parameters that can be optimized for MACD strategy
func GetMACDStrategyParameters() []Parameter {
	return []Parameter{
		{
			Name:        "fastPeriod",
			Description: "Fast period for MACD calculation",
			Default:     12,
			Min:         5,
			Max:         30,
			Step:        1,
			Type:        TypeInt,
		},
		{
			Name:        "slowPeriod",
			Description: "Slow period for MACD calculation",
			Default:     26,
			Min:         10,
			Max:         50,
			Step:        2,
			Type:        TypeInt,
		},
		{
			Name:        "signalPeriod",
			Description: "Signal period for MACD calculation",
			Default:     9,
			Min:         3,
			Max:         20,
			Step:        1,
			Type:        TypeInt,
		},
		{
			Name:        "lookbackPeriod",
			Description: "Lookback period for divergence detection",
			Default:     14,
			Min:         5,
			Max:         30,
			Step:        1,
			Type:        TypeInt,
		},
		{
			Name:        "positionSize",
			Description: "Position size as a fraction of available capital",
			Default:     0.5,
			Min:         0.1,
			Max:         1.0,
			Step:        0.1,
			Type:        TypeFloat,
		},
		{
			Name:        "stopLoss",
			Description: "Stop loss percentage",
			Default:     0.03,
			Min:         0.01,
			Max:         0.1,
			Step:        0.01,
			Type:        TypeFloat,
		},
		{
			Name:        "takeProfit",
			Description: "Take profit percentage",
			Default:     0.06,
			Min:         0.02,
			Max:         0.2,
			Step:        0.01,
			Type:        TypeFloat,
		},
	}
}
