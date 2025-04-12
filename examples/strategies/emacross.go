package strategies


type CrossEMA struct{}

func (e CrossEMA) Timeframe() string {
	return "1m"
}

func (e CrossEMA) WarmupPeriod() int {
	return 200
}

func (e CrossEMA) Indicators(df *core.Dataframe) []strategy.ChartIndicator {
	df.Metadata["ema9"] = indicator.EMA(df.Close, 9)
	df.Metadata["sma21"] = indicator.SMA(df.Close, 21)

	return []strategy.ChartIndicator{
		{
			Overlay:   tcore,
			GroupName: "MA's",
			Time:      df.Time,
			Metrics: []strategy.IndicatorMetric{
				{
					Values: df.Metadata["ema9"],
					Name:   "EMA 9",
					Color:  "red",
					Style:  strategy.StyleLine,
				},
				{
					Values: df.Metadata["sma21"],
					Name:   "SMA 21",
					Color:  "blue",
					Style:  strategy.StyleLine,
				},
			},
		},
	}
}

func (e *CrossEMA) OnCandle(df *backnrun.Dataframe, broker core.Broker) {
	closePrice := df.Close.Last(0)

	assetPosition, quotePosition, err := broker.Position(df.Pair)
	if err != nil {
		log.Error(err)
		return
	}

	if quotePosition >= 10 && // minimum quote position to trade
		df.Metadata["ema9"].Crossover(df.Metadata["sma21"]) { // trade signal (EMA9 > SMA21)

		amount := quotePosition / closePrice // calculate amount of asset to buy
		_, err := broker.CreateOrderMarket(backnrun.SideTypeBuy, df.Pair, amount)
		if err != nil {
			log.Error(err)
		}

		return
	}

	if assetPosition > 0 &&
		df.Metadata["ema9"].Crossunder(df.Metadata["sma21"]) { // trade signal (EMA9 < SMA21)

		_, err = broker.CreateOrderMarket(backnrun.SideTypeSell, df.Pair, assetPosition)
		if err != nil {
			log.Error(err)
		}
	}
}
