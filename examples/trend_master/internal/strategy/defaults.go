package strategy

// GetDefaultConfig returns the default TrendMaster configuration
func GetDefaultConfig() *TrendMasterConfig {
	config := &TrendMasterConfig{}

	// General settings
	config.General.Timeframe = "15m"
	config.General.HigherTimeframe = "1h"
	config.General.WarmupPeriod = 400
	config.General.MaxTradesPerDay = 6
	config.General.TradingHours.Enabled = true
	config.General.TradingHours.StartHour = 9
	config.General.TradingHours.EndHour = 22

	// Indicators
	config.Indicators.EMA.FastPeriod = 8
	config.Indicators.EMA.SlowPeriod = 20
	config.Indicators.EMA.LongPeriod = 50

	config.Indicators.MACD.FastPeriod = 12
	config.Indicators.MACD.SlowPeriod = 26
	config.Indicators.MACD.SignalPeriod = 9

	config.Indicators.ADX.Period = 14
	config.Indicators.ADX.Threshold = 25
	config.Indicators.ADX.MinimumDiSpread = 5

	config.Indicators.RSI.Enabled = true
	config.Indicators.RSI.Period = 14
	config.Indicators.RSI.Overbought = 70
	config.Indicators.RSI.Oversold = 30
	config.Indicators.RSI.ExtremeOverbought = 80

	config.Indicators.ATR.Period = 14
	config.Indicators.ATR.Multiplier = 2.0
	config.Indicators.ATR.VolatilityThreshold = 0.015

	config.Indicators.Volume.Enabled = true
	config.Indicators.Volume.AvgPeriod = 20
	config.Indicators.Volume.MinRatio = 1.1

	// Entry
	config.Entry.HigherTfConfirmation = true
	config.Entry.SentimentFilter.Enabled = true
	config.Entry.SentimentFilter.Threshold = 40
	config.Entry.MarketCorrelation.Enabled = true
	config.Entry.MarketCorrelation.ReferenceSymbol = "BTC"
	config.Entry.MarketCorrelation.CorrelationPeriod = 20
	config.Entry.MarketCorrelation.NegativeCorrelationThreshold = -0.5

	// Position
	config.Position.Size = 0.3
	config.Position.MaxRiskPerTrade = 0.01
	config.Position.TrailingStopPercent = 0.03
	config.Position.AdaptiveSize.Enabled = true
	config.Position.AdaptiveSize.WinIncreaseFactor = 0.1
	config.Position.AdaptiveSize.LossReductionFactor = 0.2
	config.Position.AdaptiveSize.MinSizeFactor = 0.4
	config.Position.AdaptiveSize.MaxSizeFactor = 1.5

	// Exit strategies
	config.Exit.PartialTakeProfit.Enabled = true
	config.Exit.PartialTakeProfit.Levels = []TakeProfitLevel{
		{Percentage: 0.5, Target: 0.06, TrailingOnly: false},
		{Percentage: 0.25, Target: 0.09, TrailingOnly: false},
		{Percentage: 0.25, Target: 0.0, TrailingOnly: true},
	}

	config.Exit.DynamicTargets.Enabled = true
	config.Exit.DynamicTargets.BaseTarget = 0.06
	config.Exit.DynamicTargets.AtrFactor = 3
	config.Exit.DynamicTargets.MinTarget = 0.04
	config.Exit.DynamicTargets.MaxTarget = 0.12

	config.Exit.QuickExit.MacdReversal = true
	config.Exit.QuickExit.MacdThreshold = 1.5
	config.Exit.QuickExit.AdxFalling = true
	config.Exit.QuickExit.PriceAction = true

	// Market-specific settings
	config.MarketSpecific.Crypto.VolatilityThreshold = 0.02
	config.MarketSpecific.Crypto.TrailingStopPercent = 0.04
	config.MarketSpecific.Crypto.AtrMultiplier = 2.5

	config.MarketSpecific.Forex.VolatilityThreshold = 0.008
	config.MarketSpecific.Forex.TrailingStopPercent = 0.02
	config.MarketSpecific.Forex.AtrMultiplier = 1.8

	config.MarketSpecific.Stocks.VolatilityThreshold = 0.012
	config.MarketSpecific.Stocks.TrailingStopPercent = 0.03
	config.MarketSpecific.Stocks.AtrMultiplier = 2.0

	// Logging
	config.Logging.Level = "info"
	config.Logging.TradeStatistics = true
	config.Logging.PerformanceMetrics = true

	return config
}
