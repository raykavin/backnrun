package backnrun

import (
	"os"
	"strconv"

	"github.com/raykavin/backnrun/pkg/logger/zerolog"
)

const (
	// Default configuration values
	defaultLogLevel      = "debug"
	defaultLogTimeFormat = "2006-01-02 15:04:05"
	defaultLogColored    = "true"
	defaultLogJSON       = "false"
)

// Environment variable names
const (
	envLogLevel      = "BACKNRUN_LOG_LEVEL"
	envLogTimeFormat = "BACKNRUN_LOG_TIME_FORMAT"
	envLogColor      = "BACKNRUN_LOG_COLOR"
	envLogJSON       = "BACKNRUN_LOG_JSON"
)

func init() {
	// Initialize the logger with configuration from environment variables
	log, err := initLogger()
	if err != nil {
		panic(err)
	}

	DefaultLog = zerolog.NewAdapter(log.Logger)
}

// initLogger creates a new logger instance configured from environment variables
func initLogger() (*zerolog.Logger, error) {
	// Get configuration from environment variables with defaults
	logLevel := getEnvWithDefault(envLogLevel, defaultLogLevel)
	logTimeFormat := getEnvWithDefault(envLogTimeFormat, defaultLogTimeFormat)

	// Parse boolean configurations
	logColored, err := parseBoolEnv(envLogColor, defaultLogColored)
	if err != nil {
		return nil, err
	}

	logJSON, err := parseBoolEnv(envLogJSON, defaultLogJSON)
	if err != nil {
		return nil, err
	}

	// Create and return the logger
	return zerolog.New(logLevel, logTimeFormat, logColored, logJSON)
}

// getEnvWithDefault returns the value of the environment variable or the default if not set
func getEnvWithDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// parseBoolEnv gets a boolean environment variable with a default value
func parseBoolEnv(key, defaultValue string) (bool, error) {
	value := getEnvWithDefault(key, defaultValue)
	return strconv.ParseBool(value)
}
