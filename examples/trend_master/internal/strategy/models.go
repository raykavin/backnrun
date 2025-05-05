// Package strategy implements trading strategies
package strategy

// TrendMasterConfig represents the complete configuration for the TrendMaster strategy
type TrendMasterConfig struct {
	General        GeneralConfig        `mapstructure:"general"`
	Indicators     IndicatorsConfig     `mapstructure:"indicators"`
	Entry          EntryConfig          `mapstructure:"entry"`
	Position       PositionConfig       `mapstructure:"position"`
	Exit           ExitConfig           `mapstructure:"exit"`
	MarketSpecific MarketSpecificConfig `mapstructure:"market_specific"`
	Logging        LoggingConfig        `mapstructure:"logging"`
}

// GeneralConfig contains the general trading configuration
type GeneralConfig struct {
	Timeframe       string             `mapstructure:"timeframe"`
	HigherTimeframe string             `mapstructure:"higher_timeframe"`
	WarmupPeriod    int                `mapstructure:"warmup_period"`
	MaxTradesPerDay int                `mapstructure:"max_trades_per_day"`
	TradingHours    TradingHoursConfig `mapstructure:"trading_hours"`
	Pairs           []string           `mapstructure:"pairs"`
}

// TradingHoursConfig defines trading hours constraints
type TradingHoursConfig struct {
	Enabled   bool `mapstructure:"enabled"`
	StartHour int  `mapstructure:"start_hour"`
	EndHour   int  `mapstructure:"end_hour"`
}

// IndicatorsConfig groups all technical indicators configuration
type IndicatorsConfig struct {
	EMA    EMAConfig    `mapstructure:"ema"`
	MACD   MACDConfig   `mapstructure:"macd"`
	ADX    ADXConfig    `mapstructure:"adx"`
	RSI    RSIConfig    `mapstructure:"rsi"`
	ATR    ATRConfig    `mapstructure:"atr"`
	Volume VolumeConfig `mapstructure:"volume"`
}

// EMAConfig holds Exponential Moving Average parameters
type EMAConfig struct {
	FastPeriod int `mapstructure:"fast_period"`
	SlowPeriod int `mapstructure:"slow_period"`
	LongPeriod int `mapstructure:"long_period"`
}

// MACDConfig holds MACD indicator parameters
type MACDConfig struct {
	FastPeriod   int `mapstructure:"fast_period"`
	SlowPeriod   int `mapstructure:"slow_period"`
	SignalPeriod int `mapstructure:"signal_period"`
}

// ADXConfig holds Average Directional Index parameters
type ADXConfig struct {
	Period          int     `mapstructure:"period"`
	Threshold       float64 `mapstructure:"threshold"`
	MinimumDiSpread float64 `mapstructure:"minimum_di_spread"`
}

// RSIConfig holds Relative Strength Index parameters
type RSIConfig struct {
	Enabled           bool    `mapstructure:"enabled"`
	Period            int     `mapstructure:"period"`
	Overbought        float64 `mapstructure:"overbought"`
	Oversold          float64 `mapstructure:"oversold"`
	ExtremeOverbought float64 `mapstructure:"extreme_overbought"`
}

// ATRConfig holds Average True Range parameters
type ATRConfig struct {
	Period              int     `mapstructure:"period"`
	Multiplier          float64 `mapstructure:"multiplier"`
	VolatilityThreshold float64 `mapstructure:"volatility_threshold"`
}

// VolumeConfig holds volume analysis parameters
type VolumeConfig struct {
	Enabled   bool    `mapstructure:"enabled"`
	AvgPeriod int     `mapstructure:"avg_period"`
	MinRatio  float64 `mapstructure:"min_ratio"`
}

// EntryConfig holds trade entry parameters
type EntryConfig struct {
	HigherTfConfirmation bool                    `mapstructure:"higher_tf_confirmation"`
	SentimentFilter      SentimentFilterConfig   `mapstructure:"sentiment_filter"`
	MarketCorrelation    MarketCorrelationConfig `mapstructure:"market_correlation"`
}

// SentimentFilterConfig holds market sentiment filter parameters
type SentimentFilterConfig struct {
	Enabled   bool    `mapstructure:"enabled"`
	Threshold float64 `mapstructure:"threshold"`
}

// MarketCorrelationConfig holds correlation analysis parameters
type MarketCorrelationConfig struct {
	Enabled                      bool    `mapstructure:"enabled"`
	ReferenceSymbol              string  `mapstructure:"reference_symbol"`
	CorrelationPeriod            int     `mapstructure:"correlation_period"`
	NegativeCorrelationThreshold float64 `mapstructure:"negative_correlation_threshold"`
}

// PositionConfig holds position sizing and management parameters
type PositionConfig struct {
	Size                float64            `mapstructure:"size"`
	MaxRiskPerTrade     float64            `mapstructure:"max_risk_per_trade"`
	TrailingStopPercent float64            `mapstructure:"trailing_stop_percent"`
	AdaptiveSize        AdaptiveSizeConfig `mapstructure:"adaptive_size"`
}

// AdaptiveSizeConfig holds adaptive position sizing parameters
type AdaptiveSizeConfig struct {
	Enabled             bool    `mapstructure:"enabled"`
	WinIncreaseFactor   float64 `mapstructure:"win_increase_factor"`
	LossReductionFactor float64 `mapstructure:"loss_reduction_factor"`
	MinSizeFactor       float64 `mapstructure:"min_size_factor"`
	MaxSizeFactor       float64 `mapstructure:"max_size_factor"`
}

// ExitConfig holds exit strategy parameters
type ExitConfig struct {
	PartialTakeProfit PartialTakeProfitConfig `mapstructure:"partial_take_profit"`
	DynamicTargets    DynamicTargetsConfig    `mapstructure:"dynamic_targets"`
	QuickExit         QuickExitConfig         `mapstructure:"quick_exit"`
}

// PartialTakeProfitConfig defines partial profit taking rules
type PartialTakeProfitConfig struct {
	Enabled bool              `mapstructure:"enabled"`
	Levels  []TakeProfitLevel `mapstructure:"levels"`
}

// TakeProfitLevel defines a specific profit taking level
type TakeProfitLevel struct {
	Percentage   float64 `mapstructure:"percentage"`
	Target       float64 `mapstructure:"target,omitempty"`
	TrailingOnly bool    `mapstructure:"trailing_only,omitempty"`
}

// DynamicTargetsConfig holds dynamic profit target parameters
type DynamicTargetsConfig struct {
	Enabled    bool    `mapstructure:"enabled"`
	BaseTarget float64 `mapstructure:"base_target"`
	AtrFactor  float64 `mapstructure:"atr_factor"`
	MinTarget  float64 `mapstructure:"min_target"`
	MaxTarget  float64 `mapstructure:"max_target"`
}

// QuickExitConfig holds early exit parameters
type QuickExitConfig struct {
	MacdReversal  bool    `mapstructure:"macd_reversal"`
	MacdThreshold float64 `mapstructure:"macd_threshold"`
	AdxFalling    bool    `mapstructure:"adx_falling"`
	PriceAction   bool    `mapstructure:"price_action"`
}

// MarketSpecificConfig contains market-specific settings for different asset classes
type MarketSpecificConfig struct {
	Crypto MarketTypeConfig `mapstructure:"crypto"`
	Forex  MarketTypeConfig `mapstructure:"forex"`
	Stocks MarketTypeConfig `mapstructure:"stocks"`
}

// MarketTypeConfig contains settings for a specific market type
type MarketTypeConfig struct {
	VolatilityThreshold float64 `mapstructure:"volatility_threshold"`
	TrailingStopPercent float64 `mapstructure:"trailing_stop_percent"`
	AtrMultiplier       float64 `mapstructure:"atr_multiplier"`
}

// LoggingConfig defines logging behavior
type LoggingConfig struct {
	Level              string `mapstructure:"level"`
	TradeStatistics    bool   `mapstructure:"trade_statistics"`
	PerformanceMetrics bool   `mapstructure:"performance_metrics"`
}
