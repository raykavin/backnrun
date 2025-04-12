package logger

import (
	"fmt"

	"github.com/rs/zerolog"
)

type ZerologContext struct {
	*zerolog.Logger
}

// Print implements Logger.
func (z *ZerologContext) Print(args ...any) {
	z.Logger.Print(args...)
}

// Debug implements Logger.
func (z *ZerologContext) Debug(args ...any) {
	z.Logger.Debug().Msg(fmt.Sprint(args...))
}

// Fatal implements Logger.
func (z *ZerologContext) Fatal(args ...any) {
	z.Logger.Fatal().Msg(fmt.Sprint(args...))
}

// Info implements Logger
func (z *ZerologContext) Info(args ...any) {
	z.Logger.Info().Msg(fmt.Sprint(args...))
}

// Warn implements Logger.
func (z *ZerologContext) Warn(args ...any) {
	z.Logger.Warn().Msg(fmt.Sprint(args...))
}

// Panic implements Logger.
func (z *ZerologContext) Panic(args ...any) {
	z.Logger.Panic().Msg(fmt.Sprint(args...))
}

// Infof implements Logger.
func (z *ZerologContext) Infof(format string, args ...any) {
	z.Logger.Info().Msgf(format, args...)
}

// Fatalf implements Logger.
func (z *ZerologContext) Fatalf(format string, args ...any) {
	z.Logger.Fatal().Msgf(format, args...)
}

func (z *ZerologContext) Debugf(format string, args ...any) {
	z.Logger.Debug().Msgf(format, args...)
}

// Panicf implements Logger.
func (z *ZerologContext) Panicf(format string, args ...any) {
	z.Logger.Panic().Msgf(format, args...)
}

// Printf implements Logger.
func (z *ZerologContext) Printf(format string, args ...any) {
	z.Logger.Printf(format, args...)
}

// Warnf implements Logger.
func (z *ZerologContext) Warnf(format string, args ...any) {
	z.Logger.Warn().Msgf(format, args...)
}

// Error implements Logger.
func (z *ZerologContext) Error(args ...any) {
	z.Logger.Error().Msg(fmt.Sprint(args...))
}

// Errorf implements Logger.
func (z *ZerologContext) Errorf(format string, args ...any) {
	z.Logger.Error().Msgf(format, args...)
}

// WithError implements Logger.
func (z *ZerologContext) WithError(err error) *ZerologContext {
	newLogger := z.With().Err(err).Logger()
	return &ZerologContext{&newLogger}
}

// WithField implements Logger.
func (z *ZerologContext) WithField(key string, value any) *ZerologContext {
	newLogger := z.With().Interface(key, fmt.Sprint(value)).Logger()
	return &ZerologContext{&newLogger}
}

// WithFields implements Logger.
func (z *ZerologContext) WithFields(fields map[string]any) *ZerologContext {
	newLogger := z.With().Fields(fields).Logger()

	return &ZerologContext{&newLogger}
}
