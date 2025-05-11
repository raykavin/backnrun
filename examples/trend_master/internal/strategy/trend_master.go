package strategy

import (
	"github.com/raykavin/backnrun/strategies"
)

// NewTrendMasterStrategy creates a new instance of the TrendMaster strategy
func NewTrendMasterStrategy(cfg *TrendMasterConfig) *strategies.TrendMaster {
	// Convert configuration to the strategy implementation format
	strategyConfig := ConvertToStrategyImplementation(cfg)

	// Create and return the strategy instance
	return strategies.NewTrendMaster(strategyConfig)
}

// ConvertToStrategyImplementation converts our domain model to the actual implementation
func ConvertToStrategyImplementation(cfg *TrendMasterConfig) strategies.TrendMasterConfig {
	// Create market-specific settings map
	marketSpecificSettings := make(map[string]strategies.MarketSpecificConfig)

	// Convert crypto settings
	marketSpecificSettings["crypto"] = strategies.MarketSpecificConfig{
		VolatilityThreshold: cfg.MarketSpecific.Crypto.VolatilityThreshold,
		TrailingStopPercent: cfg.MarketSpecific.Crypto.TrailingStopPercent,
		AtrMultiplier:       cfg.MarketSpecific.Crypto.AtrMultiplier,
	}

	// // Convert forex settings
	// marketSpecificSettings["forex"] = strategies.MarketSpecificConfig{
	// 	VolatilityThreshold: cfg.MarketSpecific.Forex.VolatilityThreshold,
	// 	TrailingStopPercent: cfg.MarketSpecific.Forex.TrailingStopPercent,
	// 	AtrMultiplier:       cfg.MarketSpecific.Forex.AtrMultiplier,
	// }

	// // Convert stock settings
	// marketSpecificSettings["stocks"] = strategies.MarketSpecificConfig{
	// 	VolatilityThreshold: cfg.MarketSpecific.Stocks.VolatilityThreshold,
	// 	TrailingStopPercent: cfg.MarketSpecific.Stocks.TrailingStopPercent,
	// 	AtrMultiplier:       cfg.MarketSpecific.Stocks.AtrMultiplier,
	// }

	// Convert partial exit levels
	var partialExitLevels []strategies.PartialExitLevel
	for _, level := range cfg.Exit.PartialTakeProfit.Levels {
		partialExitLevels = append(partialExitLevels, strategies.PartialExitLevel{
			Percentage:   level.Percentage,
			Target:       level.Target,
			TrailingOnly: level.TrailingOnly,
		})
	}

	// Create and return the strategy configuration
	return strategies.TrendMasterConfig{
		// Timeframe and general settings
		Timeframe:           cfg.General.Timeframe,
		HigherTimeframe:     cfg.General.HigherTimeframe,
		WarmupPeriod:        cfg.General.WarmupPeriod,
		MaxTradesPerDay:     cfg.General.MaxTradesPerDay,
		TradingHoursEnabled: cfg.General.TradingHours.Enabled,
		TradingStartHour:    cfg.General.TradingHours.StartHour,
		TradingEndHour:      cfg.General.TradingHours.EndHour,

		// EMA indicator parameters
		EmaFastPeriod: cfg.Indicators.EMA.FastPeriod,
		EmaSlowPeriod: cfg.Indicators.EMA.SlowPeriod,
		EmaLongPeriod: cfg.Indicators.EMA.LongPeriod,

		// MACD parameters
		MacdFastPeriod:   cfg.Indicators.MACD.FastPeriod,
		MacdSlowPeriod:   cfg.Indicators.MACD.SlowPeriod,
		MacdSignalPeriod: cfg.Indicators.MACD.SignalPeriod,

		// ADX parameters
		AdxPeriod:          cfg.Indicators.ADX.Period,
		AdxThreshold:       cfg.Indicators.ADX.Threshold,
		AdxMinimumDiSpread: cfg.Indicators.ADX.MinimumDiSpread,

		// RSI parameters
		UseRsiFilter:         cfg.Indicators.RSI.Enabled,
		RsiPeriod:            cfg.Indicators.RSI.Period,
		RsiOverbought:        cfg.Indicators.RSI.Overbought,
		RsiOversold:          cfg.Indicators.RSI.Oversold,
		RsiExtremeOverbought: cfg.Indicators.RSI.ExtremeOverbought,

		// ATR parameters
		AtrPeriod:           cfg.Indicators.ATR.Period,
		AtrMultiplier:       cfg.Indicators.ATR.Multiplier,
		VolatilityThreshold: cfg.Indicators.ATR.VolatilityThreshold,

		// Volume parameters
		UseVolFilter: cfg.Indicators.Volume.Enabled,
		VolAvgPeriod: cfg.Indicators.Volume.AvgPeriod,
		VolMinRatio:  cfg.Indicators.Volume.MinRatio,

		// Entry control
		UseHigherTfConfirmation: cfg.Entry.HigherTfConfirmation,

		// Sentiment filter
		UseSentimentFilter: cfg.Entry.SentimentFilter.Enabled,
		SentimentThreshold: cfg.Entry.SentimentFilter.Threshold,

		// Market correlation
		UseMarketCorrelation:         cfg.Entry.MarketCorrelation.Enabled,
		CorrelationReferenceSymbol:   cfg.Entry.MarketCorrelation.ReferenceSymbol,
		CorrelationPeriod:            cfg.Entry.MarketCorrelation.CorrelationPeriod,
		NegativeCorrelationThreshold: cfg.Entry.MarketCorrelation.NegativeCorrelationThreshold,

		// Position management
		PositionSize:        cfg.Position.Size,
		MaxRiskPerTrade:     cfg.Position.MaxRiskPerTrade,
		TrailingStopPercent: cfg.Position.TrailingStopPercent,

		// Adaptive position sizing
		UseAdaptiveSize:       cfg.Position.AdaptiveSize.Enabled,
		WinIncreaseFactor:     cfg.Position.AdaptiveSize.WinIncreaseFactor,
		LossReductionFactor:   cfg.Position.AdaptiveSize.LossReductionFactor,
		MinPositionSizeFactor: cfg.Position.AdaptiveSize.MinSizeFactor,
		MaxPositionSizeFactor: cfg.Position.AdaptiveSize.MaxSizeFactor,

		// Partial exits
		UsePartialTakeProfit: cfg.Exit.PartialTakeProfit.Enabled,
		PartialExitLevels:    partialExitLevels,

		// Dynamic targets
		UseDynamicTargets: cfg.Exit.DynamicTargets.Enabled,
		BaseTarget:        cfg.Exit.DynamicTargets.BaseTarget,
		AtrTargetFactor:   cfg.Exit.DynamicTargets.AtrFactor,
		MinTarget:         cfg.Exit.DynamicTargets.MinTarget,
		MaxTarget:         cfg.Exit.DynamicTargets.MaxTarget,

		// Quick exits
		UseMacdReversalExit:   cfg.Exit.QuickExit.MacdReversal,
		MacdReversalThreshold: cfg.Exit.QuickExit.MacdThreshold,
		UseAdxFallingExit:     cfg.Exit.QuickExit.AdxFalling,
		UsePriceActionExit:    cfg.Exit.QuickExit.PriceAction,

		// Market-specific settings
		MarketSpecificSettings: marketSpecificSettings,
	}
}
