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

// AssetQuote representa o par de ativos (moeda base e cotação)
type AssetQuote struct {
	Quote string `json:"quote"`
	Asset string `json:"asset"`
}

// PairService gerencia as informações sobre pares de trading
type PairService struct {
	pairMap map[string]AssetQuote
	mu      sync.RWMutex
}

//go:embed assets/pairs.json
var embeddedPairs []byte

// defaultPairService é a instância padrão do serviço de pares
var defaultPairService *PairService

// init inicializa o serviço de pares com dados do arquivo embutido
func init() {
	var err error
	defaultPairService, err = NewPairService(embeddedPairs)
	if err != nil {
		panic(fmt.Errorf("failed to initialize pair service: %w", err))
	}
}

// NewPairService cria uma nova instância do serviço de pares
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

// SplitAssetQuote divide um par em seus componentes asset e quote
func SplitAssetQuote(pair string) (asset string, quote string) {
	defaultPairService.mu.RLock()
	defer defaultPairService.mu.RUnlock()

	data, exists := defaultPairService.pairMap[pair]
	if !exists {
		return "", ""
	}

	return data.Asset, data.Quote
}

// GetPair retorna a informação de AssetQuote para um par
func GetPair(pair string) (AssetQuote, bool) {
	defaultPairService.mu.RLock()
	defer defaultPairService.mu.RUnlock()

	data, exists := defaultPairService.pairMap[pair]
	return data, exists
}

// UpdatePairs atualiza o mapa de pares a partir da API da Binance
func UpdatePairs(ctx context.Context) error {
	// Obtém informações do mercado spot
	spotClient := binance.NewClient("", "")
	spotInfo, err := spotClient.NewExchangeInfoService().Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to get spot exchange info: %w", err)
	}

	// Obtém informações do mercado futures
	futureClient := futures.NewClient("", "")
	futureInfo, err := futureClient.NewExchangeInfoService().Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to get futures exchange info: %w", err)
	}

	// Cria um novo mapa para armazenar os pares
	newPairMap := make(map[string]AssetQuote)

	// Adiciona informações do mercado spot
	for _, info := range spotInfo.Symbols {
		newPairMap[info.Symbol] = AssetQuote{
			Quote: info.QuoteAsset,
			Asset: info.BaseAsset,
		}
	}

	// Adiciona informações do mercado futures
	for _, info := range futureInfo.Symbols {
		newPairMap[info.Symbol] = AssetQuote{
			Quote: info.QuoteAsset,
			Asset: info.BaseAsset,
		}
	}

	// Atualiza o mapa global
	defaultPairService.mu.Lock()
	defaultPairService.pairMap = newPairMap
	defaultPairService.mu.Unlock()

	fmt.Printf("Total pairs updated: %d\n", len(newPairMap))
	return nil
}

// SavePairsToFile salva o mapa de pares em um arquivo
func SavePairsToFile(filename string) error {
	defaultPairService.mu.RLock()
	defer defaultPairService.mu.RUnlock()

	// Serializa o mapa para JSON
	content, err := json.MarshalIndent(defaultPairService.pairMap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal pairs: %w", err)
	}

	// Escreve no arquivo
	err = os.WriteFile(filename, content, 0644)
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}

// UpdateAndSavePairs atualiza e salva o mapa de pares em um arquivo
func UpdateAndSavePairs(ctx context.Context, filename string) error {
	// Atualiza o mapa de pares
	if err := UpdatePairs(ctx); err != nil {
		return err
	}

	// Salva o mapa de pares no arquivo
	return SavePairsToFile(filename)
}
