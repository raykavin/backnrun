package strategies

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/markcheno/go-talib"
	"github.com/raykavin/backnrun/bot"
	"github.com/raykavin/backnrun/core"
	"github.com/raykavin/backnrun/indicator"
	"github.com/sashabaranov/go-openai"
)

// StrategyConfig holds all configuration parameters for the ChatGPT strategy
type StrategyConfig struct {
	APIKey           string
	Model            string
	AnalysisInterval int
	Timeframe        string
	WarmupCandles    int
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() StrategyConfig {
	return StrategyConfig{
		Model:            "gpt-4-turbo-preview",
		AnalysisInterval: 1,
		Timeframe:        "5m",
		WarmupCandles:    200,
	}
}

// Position represents a trading position
type Position struct {
	InPosition bool
	EntryPrice float64
}

// MarketAnalyzer is responsible for analyzing market data
type MarketAnalyzer struct {
	client *openai.Client
	config StrategyConfig
}

// NewMarketAnalyzer creates a new market analyzer
func NewMarketAnalyzer(config StrategyConfig) *MarketAnalyzer {
	client := openai.NewClient(config.APIKey)
	return &MarketAnalyzer{
		client: client,
		config: config,
	}
}

// AnalyzeMarket sends market data to ChatGPT and gets trading signals
func (a *MarketAnalyzer) AnalyzeMarket(ctx context.Context, df *core.Dataframe, inPosition bool) (string, string, error) {
	// Prepare market data for analysis
	marketData := a.prepareMarketData(df)

	// Create the prompt for ChatGPT
	prompt := a.createAnalysisPrompt(df.Pair, marketData, inPosition)

	// Call ChatGPT API
	resp, err := a.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: a.config.Model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: a.getSystemPrompt(),
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			Temperature: 0.1, // Low temperature for more consistent responses
		},
	)

	if err != nil {
		return "", "", fmt.Errorf("ChatGPT API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", "", fmt.Errorf("empty response from ChatGPT")
	}

	// Parse the response
	content := resp.Choices[0].Message.Content

	// Extract the JSON part from the response
	jsonStr := a.extractJSON(content)
	if jsonStr == "" {
		return "", "", fmt.Errorf("could not extract JSON from response: %s", content)
	}

	// Parse the JSON
	var result struct {
		Signal    string `json:"signal"`
		Reasoning string `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return "", "", fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Validate the signal
	signal := strings.ToLower(result.Signal)
	if signal != "buy" && signal != "sell" && signal != "hold" {
		return "hold", result.Reasoning, fmt.Errorf("invalid signal: %s, defaulting to hold", signal)
	}

	return signal, result.Reasoning, nil
}

// prepareMarketData prepares market data for analysis
func (a *MarketAnalyzer) prepareMarketData(df *core.Dataframe) map[string]any {
	// Get the last 20 candles for analysis
	numCandles := 20
	if len(df.Close) < numCandles {
		numCandles = len(df.Close)
	}

	// Extract recent price data
	recentPrices := make([]float64, numCandles)
	recentVolumes := make([]float64, numCandles)
	recentTimes := make([]string, numCandles)

	for i := range numCandles {
		idx := len(df.Close) - numCandles + i
		recentPrices[i] = df.Close[idx]
		recentVolumes[i] = df.Volume[idx]
		recentTimes[i] = df.Time[idx].Format(time.RFC3339)
	}

	// Get current indicator values
	currentPrice := df.Close.Last(0)
	ema9 := df.Metadata["ema9"].Last(0)
	ema21 := df.Metadata["ema21"].Last(0)
	ema50 := df.Metadata["ema50"].Last(0)
	ema200 := df.Metadata["ema200"].Last(0)
	rsi := df.Metadata["rsi"].Last(0)
	macd := df.Metadata["macd"].Last(0)
	macdSignal := df.Metadata["macd_signal"].Last(0)
	macdHist := df.Metadata["macd_hist"].Last(0)
	bbUpper := df.Metadata["bb_upper"].Last(0)
	bbMiddle := df.Metadata["bb_middle"].Last(0)
	bbLower := df.Metadata["bb_lower"].Last(0)

	// Calculate price changes
	priceChange1h := a.calculatePriceChange(df, 12)   // 12 candles = 1 hour with 5m timeframe
	priceChange24h := a.calculatePriceChange(df, 288) // 288 candles = 24 hours with 5m timeframe

	// Create market data object
	return map[string]any{
		"current_price":  currentPrice,
		"recent_prices":  recentPrices,
		"recent_volumes": recentVolumes,
		"recent_times":   recentTimes,
		"indicators": map[string]any{
			"ema9":        ema9,
			"ema21":       ema21,
			"ema50":       ema50,
			"ema200":      ema200,
			"rsi":         rsi,
			"macd":        macd,
			"macd_signal": macdSignal,
			"macd_hist":   macdHist,
			"bollinger_bands": map[string]any{
				"upper":  bbUpper,
				"middle": bbMiddle,
				"lower":  bbLower,
			},
		},
		"price_changes": map[string]any{
			"1h":  priceChange1h,
			"24h": priceChange24h,
		},
	}
}

// calculatePriceChange calculates percentage price change over a specified number of candles
func (a *MarketAnalyzer) calculatePriceChange(df *core.Dataframe, numCandles int) float64 {
	if len(df.Close) <= numCandles {
		return 0.0
	}

	currentPrice := df.Close.Last(0)
	oldPrice := df.Close[len(df.Close)-(numCandles+1)]
	return (currentPrice - oldPrice) / oldPrice * 100
}

// createAnalysisPrompt creates the prompt for ChatGPT
func (a *MarketAnalyzer) createAnalysisPrompt(pair string, marketData map[string]any, inPosition bool) string {
	// Convert market data to JSON
	marketDataJSON, _ := json.MarshalIndent(marketData, "", "  ")

	// Create the prompt
	positionStatus := "not currently in a position"
	if inPosition {
		positionStatus = "currently in a long position"
	}

	return fmt.Sprintf(`Analyze the following market data for %s and provide a trading signal.
I am %s.

Market Data:
%s

Based on this data, provide a trading signal (buy, sell, or hold) and your reasoning.
Respond with a JSON object containing "signal" and "reasoning" fields.
`, pair, positionStatus, string(marketDataJSON))
}

// getSystemPrompt returns the system prompt for ChatGPT
func (a *MarketAnalyzer) getSystemPrompt() string {
	return `You are an expert cryptocurrency trading algorithm. Your task is to analyze market data and provide trading signals.

You should consider:
1. Technical indicators (EMA, RSI, MACD, Bollinger Bands)
2. Price action and patterns
3. Volume analysis
4. Market trends

For each analysis, provide:
1. A clear signal: "buy", "sell", or "hold"
2. A concise explanation of your reasoning

Your response must be in valid JSON format with the following structure:
{
  "signal": "buy|sell|hold",
  "reasoning": "Your detailed analysis and reasoning here"
}

Be decisive and clear in your recommendations. Focus on identifying high-probability trading opportunities while managing risk.`
}

// extractJSON extracts JSON from a string
func (a *MarketAnalyzer) extractJSON(content string) string {
	// Find the first { and the last }
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")

	if start == -1 || end == -1 || end <= start {
		return ""
	}

	return content[start : end+1]
}

// TradeExecutor is responsible for executing trades
type TradeExecutor struct {
	logger core.Logger
}

// NewTradeExecutor creates a new trade executor
func NewTradeExecutor() *TradeExecutor {
	return &TradeExecutor{
		logger: bot.DefaultLog,
	}
}

// ExecuteBuy performs the buy operation
func (e *TradeExecutor) ExecuteBuy(ctx context.Context, df *core.Dataframe, broker core.Broker,
	quotePosition, closePrice float64, reasoning string) (float64, error) {

	pair := df.Pair

	// Use 95% of available quote currency
	amount := (quotePosition * 0.95) / closePrice

	// Execute market buy
	order, err := broker.CreateOrderMarket(ctx, core.SideTypeBuy, pair, amount)
	if err != nil {
		e.logger.WithFields(map[string]any{
			"pair":  pair,
			"side":  core.SideTypeBuy,
			"price": closePrice,
			"size":  amount,
			"error": err.Error(),
		}).Error("Failed to execute buy")
		return 0, err
	}

	e.logger.WithFields(map[string]any{
		"pair":      pair,
		"price":     closePrice,
		"size":      amount,
		"order_id":  order.ID,
		"reasoning": reasoning,
	}).Info("Buy executed")

	return closePrice, nil
}

// ExecuteSell performs the sell operation
func (e *TradeExecutor) ExecuteSell(ctx context.Context, df *core.Dataframe, broker core.Broker,
	assetPosition, closePrice, entryPrice float64, reasoning string) error {

	pair := df.Pair

	// Execute market sell
	order, err := broker.CreateOrderMarket(ctx, core.SideTypeSell, pair, assetPosition)
	if err != nil {
		e.logger.WithFields(map[string]any{
			"pair":  pair,
			"side":  core.SideTypeSell,
			"price": closePrice,
			"size":  assetPosition,
			"error": err.Error(),
		}).Error("Failed to execute sell")
		return err
	}

	// Calculate profit/loss
	pnlPercent := 0.0
	if entryPrice > 0 {
		pnlPercent = (closePrice - entryPrice) / entryPrice * 100
	}

	e.logger.WithFields(map[string]any{
		"pair":      pair,
		"price":     closePrice,
		"size":      assetPosition,
		"order_id":  order.ID,
		"pnl":       pnlPercent,
		"reasoning": reasoning,
	}).Info("Sell executed")

	return nil
}

// IndicatorCalculator is responsible for calculating technical indicators
type IndicatorCalculator struct{}

// NewIndicatorCalculator creates a new indicator calculator
func NewIndicatorCalculator() *IndicatorCalculator {
	return &IndicatorCalculator{}
}

// CalculateIndicators calculates technical indicators for the dataframe
func (c *IndicatorCalculator) CalculateIndicators(df *core.Dataframe) {
	// Moving Averages
	df.Metadata["ema9"] = indicator.EMA(df.Close, 9)
	df.Metadata["ema21"] = indicator.EMA(df.Close, 21)
	df.Metadata["ema50"] = indicator.EMA(df.Close, 50)
	df.Metadata["ema200"] = indicator.EMA(df.Close, 200)

	// MACD
	df.Metadata["macd"], df.Metadata["macd_signal"], df.Metadata["macd_hist"] = indicator.MACD(
		df.Close,
		12,
		26,
		9,
	)

	// RSI
	df.Metadata["rsi"] = indicator.RSI(df.Close, 14)

	// Bollinger Bands
	df.Metadata["bb_upper"], df.Metadata["bb_middle"], df.Metadata["bb_lower"] = talib.BBands(
		df.Close,
		20,
		2.0,
		2.0,
		talib.SMA,
	)
}

// GetChartIndicators returns chart indicators for visualization
func (c *IndicatorCalculator) GetChartIndicators(df *core.Dataframe) []core.ChartIndicator {
	return []core.ChartIndicator{
		{
			Overlay:   true,
			GroupName: "Moving Averages",
			Time:      df.Time,
			Metrics: []core.IndicatorMetric{
				{
					Values: df.Metadata["ema9"],
					Name:   "EMA 9",
					Color:  "blue",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["ema21"],
					Name:   "EMA 21",
					Color:  "green",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["ema50"],
					Name:   "EMA 50",
					Color:  "purple",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["ema200"],
					Name:   "EMA 200",
					Color:  "red",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["bb_upper"],
					Name:   "BB Upper",
					Color:  "rgba(76, 175, 80, 0.5)",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["bb_middle"],
					Name:   "BB Middle",
					Color:  "rgba(76, 175, 80, 1)",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["bb_lower"],
					Name:   "BB Lower",
					Color:  "rgba(76, 175, 80, 0.5)",
					Style:  core.StyleLine,
				},
			},
		},
		{
			Overlay:   false,
			GroupName: "RSI",
			Time:      df.Time,
			Metrics: []core.IndicatorMetric{
				{
					Values: df.Metadata["rsi"],
					Name:   "RSI",
					Color:  "purple",
					Style:  core.StyleLine,
				},
			},
		},
		{
			Overlay:   false,
			GroupName: "MACD",
			Time:      df.Time,
			Metrics: []core.IndicatorMetric{
				{
					Values: df.Metadata["macd"],
					Name:   "MACD",
					Color:  "blue",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["macd_signal"],
					Name:   "Signal",
					Color:  "red",
					Style:  core.StyleLine,
				},
				{
					Values: df.Metadata["macd_hist"],
					Name:   "Histogram",
					Color:  "green",
					Style:  core.StyleHistogram,
				},
			},
		},
	}
}

// ChatGPTStrategy implements a trading strategy that uses ChatGPT
// to analyze market data and make trading decisions
type ChatGPTStrategy struct {
	config        StrategyConfig
	analyzer      *MarketAnalyzer
	executor      *TradeExecutor
	calculator    *IndicatorCalculator
	candleCounter int

	// State tracking
	lastAnalysisTime time.Time
	lastSignal       string
	lastReasoning    string
	position         Position
	logger           core.Logger
}

// NewChatGPTStrategy creates a new instance of the ChatGPTStrategy
func NewChatGPTStrategy(apiKey string) *ChatGPTStrategy {
	config := DefaultConfig()
	config.APIKey = apiKey

	return &ChatGPTStrategy{
		config:        config,
		analyzer:      NewMarketAnalyzer(config),
		executor:      NewTradeExecutor(),
		calculator:    NewIndicatorCalculator(),
		candleCounter: 0,
		lastSignal:    "hold",
		position:      Position{InPosition: false, EntryPrice: 0},
		logger:        bot.DefaultLog,
	}
}

// WithModel sets the OpenAI model to use
func (s *ChatGPTStrategy) WithModel(model string) *ChatGPTStrategy {
	s.config.Model = model
	s.analyzer = NewMarketAnalyzer(s.config)
	return s
}

// WithAnalysisInterval sets how often to perform analysis (in candles)
func (s *ChatGPTStrategy) WithAnalysisInterval(interval int) *ChatGPTStrategy {
	if interval < 1 {
		interval = 1
	}
	s.config.AnalysisInterval = interval
	return s
}

// Timeframe returns the required timeframe for this strategy
func (s *ChatGPTStrategy) Timeframe() string {
	return s.config.Timeframe
}

// WarmupPeriod returns the number of candles needed before the strategy is ready
func (s *ChatGPTStrategy) WarmupPeriod() int {
	return s.config.WarmupCandles
}

// Indicators calculates and returns the indicators used by this strategy
func (s *ChatGPTStrategy) Indicators(df *core.Dataframe) []core.ChartIndicator {
	s.calculator.CalculateIndicators(df)
	return s.calculator.GetChartIndicators(df)
}

// OnCandle is called for each new candle and implements the trading logic
func (s *ChatGPTStrategy) OnCandle(ctx context.Context, df *core.Dataframe, broker core.Broker) {
	// Get current price and positions
	closePrice := df.Close.Last(0)
	assetPosition, quotePosition, err := broker.Position(ctx, df.Pair)
	if err != nil {
		s.logger.Error(err)
		return
	}

	// Update position status
	s.position.InPosition = assetPosition > 0

	// Increment candle counter
	s.candleCounter++

	// Only analyze at specified intervals to avoid excessive API calls
	if s.candleCounter < s.config.AnalysisInterval {
		return
	}

	// Reset counter
	s.candleCounter = 0

	// Perform analysis with ChatGPT
	signal, reasoning, err := s.analyzer.AnalyzeMarket(ctx, df, s.position.InPosition)
	if err != nil {
		s.logger.WithFields(map[string]any{
			"pair":  df.Pair,
			"error": err.Error(),
		}).Error("ChatGPT analysis failed")
		return
	}

	// Store the analysis results
	s.lastAnalysisTime = time.Now()
	s.lastSignal = signal
	s.lastReasoning = reasoning

	// Log the analysis
	s.logger.WithFields(map[string]any{
		"pair":      df.Pair,
		"signal":    signal,
		"reasoning": reasoning,
	}).Info("ChatGPT analysis")

	// Execute trades based on the signal
	switch signal {
	case "buy":
		if !s.position.InPosition {
			entryPrice, err := s.executor.ExecuteBuy(ctx, df, broker, quotePosition, closePrice, reasoning)
			if err == nil {
				s.position.EntryPrice = entryPrice
				s.position.InPosition = true
			}
		}
	case "sell":
		if s.position.InPosition {
			err := s.executor.ExecuteSell(ctx, df, broker, assetPosition, closePrice, s.position.EntryPrice, reasoning)
			if err == nil {
				s.position.EntryPrice = 0
				s.position.InPosition = false
			}
		}
	case "hold":
		// Do nothing
	}
}
