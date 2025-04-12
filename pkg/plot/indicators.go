package plot

import "github.com/raykavin/backnrun/pkg/exchange"

// indicatorsByPair returns the indicators for a trading pair
func (c *Chart) indicatorsByPair(pair string) []plotIndicator {
	// Check if dataframe exists for this pair
	if _, ok := c.dataframe[pair]; !ok {
		return []plotIndicator{}
	}

	indicators := make([]plotIndicator, 0)

	// Process custom indicators
	for _, i := range c.indicators {
		i.Load(c.dataframe[pair])
		indicator := plotIndicator{
			Name:    i.Name(),
			Overlay: i.Overlay(),
			Warmup:  i.Warmup(),
			Metrics: make([]indicatorMetric, 0),
		}

		for _, metric := range i.Metrics() {
			indicator.Metrics = append(indicator.Metrics, indicatorMetric{
				Name:   metric.Name,
				Values: metric.Values,
				Time:   metric.Time,
				Color:  metric.Color,
				Style:  metric.Style,
			})
		}

		indicators = append(indicators, indicator)
	}

	// Process strategy indicators if available
	if c.strategy != nil {
		warmup := c.strategy.WarmupPeriod()
		strategyIndicators := c.strategy.Indicators(c.dataframe[pair])

		for _, i := range strategyIndicators {
			indicator := plotIndicator{
				Name:    i.GroupName,
				Overlay: i.Overlay,
				Warmup:  i.Warmup,
				Metrics: make([]indicatorMetric, 0),
			}

			for _, metric := range i.Metrics {
				if len(metric.Values) < warmup {
					continue
				}

				indicator.Metrics = append(indicator.Metrics, indicatorMetric{
					Time:   i.Time[i.Warmup:],
					Values: metric.Values[i.Warmup:],
					Name:   metric.Name,
					Color:  metric.Color,
					Style:  string(metric.Style),
				})
			}
			indicators = append(indicators, indicator)
		}
	}

	return indicators
}

// equityValuesByPair returns asset and equity values for a trading pair
func (c *Chart) equityValuesByPair(pair string) ([]AssetValue, []AssetValue) {
	assetValues := make([]AssetValue, 0)
	equityValues := make([]AssetValue, 0)

	if c.paperWallet != nil {
		asset, _ := exchange.SplitAssetQuote(pair)

		// Get asset value history
		for _, value := range c.paperWallet.AssetValues(asset) {
			assetValues = append(assetValues, AssetValue{
				Time:  value.Time,
				Value: value.Value,
			})
		}

		// Get equity value history
		for _, value := range c.paperWallet.EquityValues() {
			equityValues = append(equityValues, AssetValue{
				Time:  value.Time,
				Value: value.Value,
			})
		}
	}

	return assetValues, equityValues
}
