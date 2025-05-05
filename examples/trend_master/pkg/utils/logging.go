// Package utils provides utility functions for the application
package utils

import (
	"sync"

	"github.com/raykavin/backnrun/bot"
	"github.com/raykavin/backnrun/core"
)

var (
	logger     core.Logger
	loggerOnce sync.Once
)

// GetLogger returns a singleton logger instance
func GetLogger() core.Logger {
	loggerOnce.Do(func() {
		logger = bot.DefaultLog
	})
	return logger
}

// SetLogLevel sets the logging level based on configuration
func SetLogLevel(level string) {
	bot.DefaultLog.SetLevel(core.DebugLevel)
}
