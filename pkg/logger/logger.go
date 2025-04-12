package logger

type Level int8

const (
	Disabled   Level = -1   // Disabled is used for disabled logging.
	TraceLevel Level = iota // TraceLevel is used for detailed debugging information.
	DebugLevel              // DebugLevel is used for debugging information.
	InfoLevel               // InfoLevel is used for informational messages.
	WarnLevel               // WarnLevel is used for warning messages.
	ErrorLevel              // ErrorLevel is used for error messages.
	FatalLevel              // FatalLevel is used for fatal messages that cause the program to exit.
	PanicLevel              // PanicLevel is used for panic messages that cause the program to panic.
	NoLevel                 // NoLevel is used for no logging level.
)

type Logger interface {
	// Returns a logger based off the root logger and decorates it with the given context and arguments.
	WithField(key string, value any) Logger  // WithField returns a logger with the given key-value pair.
	WithFields(fields map[string]any) Logger // WithFields returns a logger with the given fields.
	WithError(err error) Logger              // WithError returns a logger with the given error.

	// Default log functions
	Print(args ...any) // Print logs the message with the default level.
	Trace(args ...any) // Trace logs the message with the trace level.
	Debug(args ...any) // Debug logs the message with the debug level.
	Info(args ...any)  // Info logs the message with the info level.
	Warn(args ...any)  // Warn logs the message with the warning level.
	Error(args ...any) // Error logs the message with the error level.
	Fatal(args ...any) // Fatal logs the message and then exits the program.
	Panic(args ...any) // Panic logs the message and then panics.

	// Log functions with format
	Printf(format string, args ...any) // Printf formats and logs the message with the given format and arguments.
	Tracef(format string, args ...any) // Tracef formats and logs the message with the given format and arguments.
	Debugf(format string, args ...any) // Debugf formats and logs the message with the given format and arguments.
	Infof(format string, args ...any)  // Infof formats and logs the message with the given format and arguments.
	Warnf(format string, args ...any)  // Warnf formats and logs the message with the given format and arguments.
	Errorf(format string, args ...any) // Errorf formats and logs the message with the given format and arguments.
	Fatalf(format string, args ...any) // Fatalf formats and logs the message with the given format and arguments.
	Panicf(format string, args ...any) // Panicf formats and logs the message with the given format and arguments.

	// Log level functions
	SetLevel(level Level) // SetLevel sets the logging level for the logger.
	GetLevel() Level      // GetLevel returns the logging level for the logger.
}
