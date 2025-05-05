// Package config handles application configuration management using Viper
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/raykavin/backnrun/examples/trend_master/internal/strategy"
	"github.com/raykavin/backnrun/examples/trend_master/pkg/utils"
	"github.com/spf13/viper"
)

// Constants for configuration
const (
	DefaultConfigPath  = "./trend_master.yaml"
	DefaultStoragePath = "./backnrun.sqlite"
)

var (
	log = utils.GetLogger()
)

// AppConfig holds the application configuration
type AppConfig struct {
	Binance     BinanceConfig
	Telegram    TelegramConfig
	ConfigPath  string
	StoragePath string
}

// BinanceConfig holds Binance exchange configuration
type BinanceConfig struct {
	APIKey     string
	SecretKey  string
	UseTestnet bool
	Debug      bool
}

// TelegramConfig holds Telegram notification configuration
type TelegramConfig struct {
	Enabled bool
	Token   string
	UserID  int
}

// LoadAppConfig loads application configuration using Viper
func LoadAppConfig() (*AppConfig, error) {
	// Set up Viper for environment variables
	viper.AutomaticEnv()

	// Set default values
	viper.SetDefault("CONFIG_PATH", DefaultConfigPath)
	viper.SetDefault("STORAGE_PATH", DefaultStoragePath)
	viper.SetDefault("BINANCE_USE_TESTNET", false)
	viper.SetDefault("BINANCE_DEBUG_CLIENT", false)
	viper.SetDefault("TELEGRAM_ENABLED", false)

	// Create the configuration
	config := &AppConfig{
		Binance: BinanceConfig{
			APIKey:     viper.GetString("BINANCE_API_KEY"),
			SecretKey:  viper.GetString("BINANCE_SECRET_KEY"),
			UseTestnet: viper.GetBool("BINANCE_USE_TESTNET"),
			Debug:      viper.GetBool("BINANCE_DEBUG_CLIENT"),
		},
		Telegram: TelegramConfig{
			Enabled: viper.GetBool("TELEGRAM_ENABLED"),
			Token:   viper.GetString("TELEGRAM_TOKEN"),
			UserID:  viper.GetInt("TELEGRAM_USER"),
		},
		ConfigPath:  viper.GetString("CONFIG_PATH"),
		StoragePath: viper.GetString("STORAGE_PATH"),
	}

	return config, nil
}

// LoadStrategyConfig loads the TrendMaster strategy configuration
func LoadStrategyConfig(configPath string) (*strategy.TrendMasterConfig, error) {
	// Check if the file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return saveDefaultConfig(configPath)
	}

	// Set up viper to read from the config file
	v := viper.New()
	v.SetConfigFile(configPath)

	// Read the configuration
	if err := v.ReadInConfig(); err != nil {
		log.WithFields(map[string]any{
			"error": err,
			"path":  configPath,
		}).Error("Failed to load configuration file")
		return saveDefaultConfig(configPath)
	}

	// Build the strategy configuration
	strategyConfig := &strategy.TrendMasterConfig{}

	// Unmarshal the configuration
	if err := v.Unmarshal(strategyConfig); err != nil {
		log.WithFields(map[string]any{
			"error": err,
		}).Error("Failed to parse configuration")
		return saveDefaultConfig(configPath)
	}

	return strategyConfig, nil
}

// saveDefaultConfig creates a default configuration file and returns the default config
func saveDefaultConfig(configPath string) (*strategy.TrendMasterConfig, error) {
	// Create directory for the default configuration file if it doesn't exist
	configDir := filepath.Dir(configPath)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		err = os.MkdirAll(configDir, 0755)
		if err != nil {
			log.Error("Could not create configuration directory")
			return strategy.GetDefaultConfig(), nil
		}
	}

	// Get default configuration
	defaultConfig := strategy.GetDefaultConfig()

	// Create a new viper instance for saving
	v := viper.New()

	// Convert struct to map
	v.Set("general", defaultConfig.General)
	v.Set("indicators", defaultConfig.Indicators)
	v.Set("entry", defaultConfig.Entry)
	v.Set("position", defaultConfig.Position)
	v.Set("exit", defaultConfig.Exit)
	v.Set("market_specific", defaultConfig.MarketSpecific)
	v.Set("logging", defaultConfig.Logging)

	// Set the configuration file
	v.SetConfigFile(configPath)

	// Save the configuration
	if err := v.WriteConfig(); err != nil {
		return defaultConfig, fmt.Errorf("could not save default configuration: %w", err)
	}

	log.WithFields(map[string]any{
		"path": configPath,
	}).Info("Default configuration file created")

	return defaultConfig, nil
}
