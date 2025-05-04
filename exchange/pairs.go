// Package exchange provides functionality for interacting with cryptocurrency exchanges
// and managing trading pair information.
package exchange

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
)

// ---------------------
// Types
// ---------------------

// AssetQuote represents a trading pair (base asset and quote currency)
type AssetQuote struct {
	Quote string `json:"quote"`
	Asset string `json:"asset"`
}

// PairService manages information about trading pairs
type PairService struct {
	pairMap map[string]AssetQuote
	mu      sync.RWMutex
}

// ---------------------
// Embedded Data
// ---------------------

//go:embed assets/pairs.json
var embeddedPairs []byte

// defaultPairService is the default instance of the pair service
var defaultPairService *PairService

// ---------------------
// Initialization
// ---------------------

// init initializes the pair service with data from the embedded file
func init() {
	var err error
	defaultPairService, err = NewPairService(embeddedPairs)
	if err != nil {
		panic(fmt.Errorf("failed to initialize pair service: %w", err))
	}
}

// NewPairService creates a new instance of the pair service
func NewPairService(pairsData []byte) (*PairService, error) {
	service := &PairService{
		pairMap: make(map[string]AssetQuote),
	}

	if len(pairsData) > 0 {
		if err := json.Unmarshal(pairsData, &service.pairMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal pairs data: %w", err)
		}
	}

	return service, nil
}

// ---------------------
// Pair Lookup Methods
// ---------------------

// SplitAssetQuote splits a pair into its asset and quote components
func SplitAssetQuote(pair string) (asset string, quote string) {
	defaultPairService.mu.RLock()
	defer defaultPairService.mu.RUnlock()

	data, exists := defaultPairService.pairMap[pair]
	if !exists {
		return "", ""
	}

	return data.Asset, data.Quote
}

// GetPair returns the AssetQuote information for a pair
func GetPair(pair string) (AssetQuote, bool) {
	defaultPairService.mu.RLock()
	defer defaultPairService.mu.RUnlock()

	data, exists := defaultPairService.pairMap[pair]
	return data, exists
}

// ---------------------
// Pair Update Methods
// ---------------------

// UpdatePairs updates the pair map from the Binance API
func UpdatePairs(ctx context.Context) error {
	// Get spot market information
	spotClient := binance.NewClient("", "")
	spotInfo, err := spotClient.NewExchangeInfoService().Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to get spot exchange info: %w", err)
	}

	// Get futures market information
	futureClient := futures.NewClient("", "")
	futureInfo, err := futureClient.NewExchangeInfoService().Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to get futures exchange info: %w", err)
	}

	// Create a new map to store the pairs
	newPairMap := make(map[string]AssetQuote)

	// Add spot market information
	for _, info := range spotInfo.Symbols {
		newPairMap[info.Symbol] = AssetQuote{
			Quote: info.QuoteAsset,
			Asset: info.BaseAsset,
		}
	}

	// Add futures market information
	for _, info := range futureInfo.Symbols {
		newPairMap[info.Symbol] = AssetQuote{
			Quote: info.QuoteAsset,
			Asset: info.BaseAsset,
		}
	}

	// Update the global map
	defaultPairService.mu.Lock()
	defaultPairService.pairMap = newPairMap
	defaultPairService.mu.Unlock()

	fmt.Printf("Total pairs updated: %d\n", len(newPairMap))
	return nil
}

// SavePairsToFile saves the pair map to a file
func SavePairsToFile(filename string) error {
	defaultPairService.mu.RLock()
	defer defaultPairService.mu.RUnlock()

	// Serialize the map to JSON
	content, err := json.MarshalIndent(defaultPairService.pairMap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal pairs: %w", err)
	}

	// Write to file
	err = os.WriteFile(filename, content, 0644)
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}

// UpdateAndSavePairs updates and saves the pair map to a file
func UpdateAndSavePairs(ctx context.Context, filename string) error {
	// Update the pair map
	if err := UpdatePairs(ctx); err != nil {
		return err
	}

	// Save the pair map to the file
	return SavePairsToFile(filename)
}
