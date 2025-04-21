package zerolog

import (
	"fmt"

	"github.com/raykavin/backnrun/core"

	"github.com/rs/zerolog"
)

type ZerologAdapter struct {
	*zerolog.Logger
}

func NewAdapter(logger *zerolog.Logger) *ZerologAdapter {
	return &ZerologAdapter{logger}
}

// GetLevel implements core.Logger.
func (z *ZerologAdapter) GetLevel() core.Level {
	return toLevel(z.Logger.GetLevel())
}

// SetLevel implements core.Logger.
func (z *ZerologAdapter) SetLevel(level core.Level) {
	zerolog.SetGlobalLevel(toZerologLevel(level))
}

// Trace implements core.Logger.
func (z *ZerologAdapter) Trace(args ...any) {
	z.Logger.Trace().Msg(fmt.Sprint(args...))
}

// Tracef implements core.Logger.
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
func (z *ZerologAdapter) WithError(err error) core.Logger {
	newLogger := z.With().Err(err).Logger()
	return &ZerologAdapter{&newLogger}
}

// WithField implements Logger.
func (z *ZerologAdapter) WithField(key string, value any) core.Logger {
	newLogger := z.With().Interface(key, fmt.Sprint(value)).Logger()
	return &ZerologAdapter{&newLogger}
}

// WithFields implements Logger.
func (z *ZerologAdapter) WithFields(fields map[string]any) core.Logger {
	newLogger := z.With().Fields(fields).Logger()

	return &ZerologAdapter{&newLogger}
}

// toLevel converts zerolog.Level to core.Level.
func toLevel(level zerolog.Level) core.Level {
	levelMap := map[zerolog.Level]core.Level{
		zerolog.Disabled:   core.Disabled,
		zerolog.NoLevel:    core.NoLevel,
		zerolog.TraceLevel: core.TraceLevel,
		zerolog.DebugLevel: core.DebugLevel,
		zerolog.InfoLevel:  core.InfoLevel,
		zerolog.WarnLevel:  core.WarnLevel,
		zerolog.ErrorLevel: core.ErrorLevel,
		zerolog.FatalLevel: core.FatalLevel,
		zerolog.PanicLevel: core.PanicLevel,
	}

	if level, ok := levelMap[level]; ok {
		return level
	}

	return core.NoLevel
}

// toZerologLevel converts core.Level to zerolog.Level.
func toZerologLevel(level core.Level) zerolog.Level {
	levelMap := map[core.Level]zerolog.Level{
		core.Disabled:   zerolog.Disabled,
		core.NoLevel:    zerolog.NoLevel,
		core.TraceLevel: zerolog.TraceLevel,
		core.DebugLevel: zerolog.DebugLevel,
		core.InfoLevel:  zerolog.InfoLevel,
		core.WarnLevel:  zerolog.WarnLevel,
		core.ErrorLevel: zerolog.ErrorLevel,
		core.FatalLevel: zerolog.FatalLevel,
		core.PanicLevel: zerolog.PanicLevel,
	}

	if level, ok := levelMap[level]; ok {
		return level
	}

	return zerolog.NoLevel
}
