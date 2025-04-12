package zerolog

import (
	"fmt"

	"github.com/raykavin/backnrun/pkg/logger"
	"github.com/rs/zerolog"
)

type ZerologAdapter struct {
	*zerolog.Logger
}

func NewAdapter(logger *zerolog.Logger) *ZerologAdapter {
	return &ZerologAdapter{logger}
}

// GetLevel implements logger.Logger.
func (z *ZerologAdapter) GetLevel() logger.Level {
	return toLevel(z.Logger.GetLevel())
}

// SetLevel implements logger.Logger.
func (z *ZerologAdapter) SetLevel(level logger.Level) {
	zerolog.SetGlobalLevel(toZerologLevel(level))
}

// Trace implements logger.Logger.
func (z *ZerologAdapter) Trace(args ...any) {
	z.Logger.Trace().Msg(fmt.Sprint(args...))
}

// Tracef implements logger.Logger.
func (z *ZerologAdapter) Tracef(format string, args ...any) {
	z.Logger.Trace().Msgf(format, args...)
}

// Print implements Logger.
func (z *ZerologAdapter) Print(args ...any) {
	z.Logger.Print(args...)
}

// Debug implements Logger.
func (z *ZerologAdapter) Debug(args ...any) {
	z.Logger.Debug().Msg(fmt.Sprint(args...))
}

// Fatal implements Logger.
func (z *ZerologAdapter) Fatal(args ...any) {
	z.Logger.Fatal().Msg(fmt.Sprint(args...))
}

// Info implements Logger
func (z *ZerologAdapter) Info(args ...any) {
	z.Logger.Info().Msg(fmt.Sprint(args...))
}

// Warn implements Logger.
func (z *ZerologAdapter) Warn(args ...any) {
	z.Logger.Warn().Msg(fmt.Sprint(args...))
}

// Panic implements Logger.
func (z *ZerologAdapter) Panic(args ...any) {
	z.Logger.Panic().Msg(fmt.Sprint(args...))
}

// Infof implements Logger.
func (z *ZerologAdapter) Infof(format string, args ...any) {
	z.Logger.Info().Msgf(format, args...)
}

// Fatalf implements Logger.
func (z *ZerologAdapter) Fatalf(format string, args ...any) {
	z.Logger.Fatal().Msgf(format, args...)
}

func (z *ZerologAdapter) Debugf(format string, args ...any) {
	z.Logger.Debug().Msgf(format, args...)
}

// Panicf implements Logger.
func (z *ZerologAdapter) Panicf(format string, args ...any) {
	z.Logger.Panic().Msgf(format, args...)
}

// Printf implements Logger.
func (z *ZerologAdapter) Printf(format string, args ...any) {
	z.Logger.Printf(format, args...)
}

// Warnf implements Logger.
func (z *ZerologAdapter) Warnf(format string, args ...any) {
	z.Logger.Warn().Msgf(format, args...)
}

// Error implements Logger.
func (z *ZerologAdapter) Error(args ...any) {
	z.Logger.Error().Msg(fmt.Sprint(args...))
}

// Errorf implements Logger.
func (z *ZerologAdapter) Errorf(format string, args ...any) {
	z.Logger.Error().Msgf(format, args...)
}

// WithError implements Logger.
func (z *ZerologAdapter) WithError(err error) logger.Logger {
	newLogger := z.With().Err(err).Logger()
	return &ZerologAdapter{&newLogger}
}

// WithField implements Logger.
func (z *ZerologAdapter) WithField(key string, value any) logger.Logger {
	newLogger := z.With().Interface(key, fmt.Sprint(value)).Logger()
	return &ZerologAdapter{&newLogger}
}

// WithFields implements Logger.
func (z *ZerologAdapter) WithFields(fields map[string]any) logger.Logger {
	newLogger := z.With().Fields(fields).Logger()

	return &ZerologAdapter{&newLogger}
}

// toLevel converts zerolog.Level to logger.Level.
func toLevel(level zerolog.Level) logger.Level {
	levelMap := map[zerolog.Level]logger.Level{
		zerolog.Disabled:   logger.Disabled,
		zerolog.NoLevel:    logger.NoLevel,
		zerolog.TraceLevel: logger.TraceLevel,
		zerolog.DebugLevel: logger.DebugLevel,
		zerolog.InfoLevel:  logger.InfoLevel,
		zerolog.WarnLevel:  logger.WarnLevel,
		zerolog.ErrorLevel: logger.ErrorLevel,
		zerolog.FatalLevel: logger.FatalLevel,
		zerolog.PanicLevel: logger.PanicLevel,
	}

	if level, ok := levelMap[level]; ok {
		return level
	}

	return logger.NoLevel
}

// toZerologLevel converts logger.Level to zerolog.Level.
func toZerologLevel(level logger.Level) zerolog.Level {
	levelMap := map[logger.Level]zerolog.Level{
		logger.Disabled:   zerolog.Disabled,
		logger.NoLevel:    zerolog.NoLevel,
		logger.TraceLevel: zerolog.TraceLevel,
		logger.DebugLevel: zerolog.DebugLevel,
		logger.InfoLevel:  zerolog.InfoLevel,
		logger.WarnLevel:  zerolog.WarnLevel,
		logger.ErrorLevel: zerolog.ErrorLevel,
		logger.FatalLevel: zerolog.FatalLevel,
		logger.PanicLevel: zerolog.PanicLevel,
	}

	if level, ok := levelMap[level]; ok {
		return level
	}

	return zerolog.NoLevel
}
