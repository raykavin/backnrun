package core

// Balance represents the available funds for a specific asset
type Balance struct {
	Asset    string
	Free     float64
	Lock     float64
	Leverage float64
}

// GetAsset returns the asset identifier
func (b Balance) GetAsset() string { return b.Asset }

// GetFree returns the amount of the asset that is available for trading
func (b Balance) GetFree() float64 { return b.Free }

// GetLock returns the amount of the asset that is locked (unavailable for trading)
func (b Balance) GetLock() float64 { return b.Lock }

// GetLeverage returns the leverage value for the asset
func (b Balance) GetLeverage() float64 { return b.Leverage }
