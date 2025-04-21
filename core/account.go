package core

import "fmt"

// Account represents a trading account with multiple asset balances
type Account struct {
	Balances []Balance
}

func NewAccount(balances []Balance) (Account, error) {
	if len(balances) == 0 {
		return Account{}, fmt.Errorf("invalid account balances")
	}

	return Account{Balances: balances}, nil
}

// GetBalances returns all balances in the account
func (a Account) GetBalances() []Balance {
	return a.Balances
}

// GetBalanceMap returns a map of asset tickers to balances for efficient lookups
func (a Account) GetBalanceMap() map[string]Balance {
	balanceMap := make(map[string]Balance, len(a.Balances))
	for _, balance := range a.Balances {
		balanceMap[balance.Asset] = balance
	}
	return balanceMap
}

// GetBalance retrieves the balance for a specific asset and quote pair
// Returns two Balance objects: one for the asset and one for the quote
// If a balance is not found for either ticker, an empty Balance is returned
func (a Account) GetBalance(assetTick, quoteTick string) (Balance, Balance) {
	var assetBalance, quoteBalance Balance
	var isSetAsset, isSetQuote bool

	// Create a map for more efficient lookups when there are many balances
	if len(a.Balances) > 10 {
		balanceMap := a.GetBalanceMap()

		if asset, ok := balanceMap[assetTick]; ok {
			assetBalance = asset
		}

		if quote, ok := balanceMap[quoteTick]; ok {
			quoteBalance = quote
		}

		return assetBalance, quoteBalance
	}

	// Use linear search for small balance lists
	for _, balance := range a.Balances {
		switch balance.Asset {
		case assetTick:
			assetBalance = balance
			isSetAsset = true
		case quoteTick:
			quoteBalance = balance
			isSetQuote = true
		}

		if isSetAsset && isSetQuote {
			break
		}
	}

	return assetBalance, quoteBalance
}

// GetEquity calculates the total equity in the account
// Equity is defined as the sum of free and locked amounts across all assets
func (a Account) GetEquity() float64 {
	var total float64

	for _, balance := range a.Balances {
		total += balance.Free + balance.Lock
	}

	return total
}
