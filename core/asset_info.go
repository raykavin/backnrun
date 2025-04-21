package core

// AssetInfo contains market information about a trading pair
type AssetInfo struct {
	BaseAsset          string
	QuoteAsset         string
	MinPrice           float64
	MaxPrice           float64
	MinQuantity        float64
	MaxQuantity        float64
	StepSize           float64
	TickSize           float64
	QuotePrecision     int
	BaseAssetPrecision int
}

// NewAssetInfo creates a new AssetInfo instance with validation
func NewAssetInfo(
	baseAsset string,
	quoteAsset string,
	minPrice float64,
	maxPrice float64,
	minQuantity float64,
	maxQuantity float64,
	stepSize float64,
	tickSize float64,
	quotePrecision int,
	baseAssetPrecision int,
) (AssetInfo, error) {
	assetInfo := AssetInfo{
		BaseAsset:          baseAsset,
		QuoteAsset:         quoteAsset,
		MinPrice:           minPrice,
		MaxPrice:           maxPrice,
		MinQuantity:        minQuantity,
		MaxQuantity:        maxQuantity,
		StepSize:           stepSize,
		TickSize:           tickSize,
		QuotePrecision:     quotePrecision,
		BaseAssetPrecision: baseAssetPrecision,
	}

	return assetInfo, assetInfo.validate()
}

// GetBaseAsset returns the base asset of the trading pair
func (a AssetInfo) GetBaseAsset() string { return a.BaseAsset }

// GetQuoteAsset returns the quote asset of the trading pair
func (a AssetInfo) GetQuoteAsset() string { return a.QuoteAsset }

// GetQuotePrecision returns the precision of the quote asset
func (a AssetInfo) GetQuotePrecision() int { return a.QuotePrecision }

// GetBaseAssetPrecision returns the precision of the base asset
func (a AssetInfo) GetBaseAssetPrecision() int { return a.BaseAssetPrecision }

// GetMinPrice returns the minimum price allowed for the trading pair
func (a AssetInfo) GetMinPrice() float64 { return a.MinPrice }

// GetMaxPrice returns the maximum price allowed for the trading pair
func (a AssetInfo) GetMaxPrice() float64 { return a.MaxPrice }

// GetTickSize returns the tick size for price increments
func (a AssetInfo) GetTickSize() float64 { return a.TickSize }

// GetMinQuantity returns the minimum quantity allowed for the trading pair
func (a AssetInfo) GetMinQuantity() float64 { return a.MinQuantity }

// GetMaxQuantity returns the maximum quantity allowed for the trading pair
func (a AssetInfo) GetMaxQuantity() float64 { return a.MaxQuantity }

// GetStepSize returns the step size for quantity increments
func (a AssetInfo) GetStepSize() float64 { return a.StepSize }

// ChangeMinPrice updates the minimum price with validation
func (a *AssetInfo) ChangeMinPrice(price float64) error {
	if isNegative(price) {
		return ErrNegativeValue
	}
	a.MinPrice = price
	return nil
}

// ChangeMaxPrice updates the maximum price with validation
func (a *AssetInfo) ChangeMaxPrice(price float64) error {
	if isNegative(price) {
		return ErrNegativeValue
	}
	a.MaxPrice = price
	return nil
}

// ChangeTickSize updates the tick size with validation
func (a *AssetInfo) ChangeTickSize(size float64) error {
	if isNegative(size) {
		return ErrNegativeValue
	}
	a.TickSize = size
	return nil
}

// ChangeMinQuantity updates the minimum quantity with validation
func (a *AssetInfo) ChangeMinQuantity(quantity float64) error {
	if isNegative(quantity) {
		return ErrNegativeValue
	}
	a.MinQuantity = quantity
	return nil
}

// ChangeMaxQuantity updates the maximum quantity with validation
func (a *AssetInfo) ChangeMaxQuantity(quantity float64) error {
	if isNegative(quantity) {
		return ErrNegativeValue
	}
	a.MaxQuantity = quantity
	return nil
}

// ChangeStepSize updates the step size with validation
func (a *AssetInfo) ChangeStepSize(size float64) error {
	if isNegative(size) {
		return ErrNegativeValue
	}
	a.StepSize = size
	return nil
}

// validate ensures that the AssetInfo has valid base and quote assets
func (a AssetInfo) validate() error {
	if len(a.BaseAsset) == 0 {
		return ErrBaseAssetEmpty
	}

	if len(a.QuoteAsset) == 0 {
		return ErrQuoteAssetEmpty
	}
	return nil
}
