package core

// AssetInfo contains market information about a trading pair
type AssetInfo struct {
	BaseAsset  string
	QuoteAsset string

	MinPrice    float64
	MaxPrice    float64
	MinQuantity float64
	MaxQuantity float64
	StepSize    float64
	TickSize    float64

	QuotePrecision     int
	BaseAssetPrecision int
}

// GetBaseAsset returns the base asset of the trading pair
func (a AssetInfo) GetBaseAsset() string { return a.BaseAsset }

// GetQuoteAsset returns the quote asset of the trading pair
func (a AssetInfo) GetQuoteAsset() string { return a.QuoteAsset }

// GetMinPrice returns the minimum price allowed for the trading pair
func (a AssetInfo) GetMinPrice() float64 { return a.MinPrice }

// GetMaxPrice returns the maximum price allowed for the trading pair
func (a AssetInfo) GetMaxPrice() float64 { return a.MaxPrice }

// GetMinQuantity returns the minimum quantity allowed for the trading pair
func (a AssetInfo) GetMinQuantity() float64 { return a.MinQuantity }

// GetMaxQuantity returns the maximum quantity allowed for the trading pair
func (a AssetInfo) GetMaxQuantity() float64 { return a.MaxQuantity }

// GetStepSize returns the step size for quantity increments
func (a AssetInfo) GetStepSize() float64 { return a.StepSize }

// GetTickSize returns the tick size for price increments
func (a AssetInfo) GetTickSize() float64 { return a.TickSize }

// GetQuotePrecision returns the precision of the quote asset
func (a AssetInfo) GetQuotePrecision() int { return a.QuotePrecision }

// GetBaseAssetPrecision returns the precision of the base asset
func (a AssetInfo) GetBaseAssetPrecision() int { return a.BaseAssetPrecision }
