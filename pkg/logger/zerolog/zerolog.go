package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/goterm/term"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

type ZerologLogger struct {
	*zerolog.Logger
}

func NewZerolog(level, dateTimeLayout string, colored, jsonFormat bool) (*ZerologLogger, error) {
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	logMode, err := zerolog.ParseLevel(level)
	if err != nil {
		return nil, err
	}

	zerolog.SetGlobalLevel(logMode)

	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		NoColor:    !colored,
		TimeFormat: dateTimeLayout,
	}

	if !jsonFormat {
		output.FormatLevel = formatLevel
		output.FormatMessage = formatMessage
		output.FormatCaller = formatCaller
		output.FormatTimestamp = func(i interface{}) string {
			return formatTimestamp(i, dateTimeLayout)
		}
	}

	logger := log.
		Output(output).
		With().
		CallerWithSkipFrameCount(3).
		Logger()

	return &ZerologLogger{&logger}, nil
}

func formatLevel(i interface{}) string {
	levelStr, ok := i.(string)
	if !ok {
		return "UNKNOWN"
	}

	levelColor := getLevelColor(levelStr)
	return levelColor
}

func getLevelColor(level string) string {
	switch level {
	case zerolog.LevelTraceValue:
		return term.Cyanf("[TRC]")
	case zerolog.LevelDebugValue:
		return term.Cyanf("[DBG]")
	case zerolog.LevelInfoValue:
		return term.Greenf("[INF]")
	case zerolog.LevelWarnValue:
		return term.Yellowf("[WAR]")
	case zerolog.LevelPanicValue:
		return term.Redf("[PAN]")
	case zerolog.LevelFatalValue:
		return term.Redf("[FTL]")
	case zerolog.LevelErrorValue:
		return term.Redf("[ERR]")
	default:
		return term.Whitef("[UNK]")
	}
}

func formatMessage(i interface{}) string {
	const maxSize = 80

	msg, ok := i.(string)
	if !ok || len(msg) == 0 {
		return ">"
	}

	// Truncate message ifis greaten of max size
	if len(msg) > maxSize {
		msg = msg[:maxSize]
	}

	if len(msg) < maxSize {
		msg += strings.Repeat(" ", maxSize-len(msg))
	}

	return term.Whitef("> %s", msg)
}

func formatCaller(i interface{}) string {
	const maxFileSize = 18
	const maxLineSize = 4

	fname, ok := i.(string)
	if !ok || len(fname) == 0 {
		return ""
	}

	caller := filepath.Base(fname)
	callerSplit := strings.Split(caller, ":")
	if len(callerSplit) != 2 {
		return caller
	}

	fileBase := callerSplit[0]
	line := callerSplit[1]

	// Truncate or pad the fileBase to ensure it has maxFileSize length
	if len(fileBase) > maxFileSize {
		fileBase = fileBase[:maxFileSize]
	} else {
		fileBase = fmt.Sprintf("%-*s", maxFileSize, fileBase)
	}

	// Ensure line number has a fixed size (truncate left if necessary)
	if len(line) > maxLineSize {
		line = line[len(line)-maxLineSize:]
	} else {
		line = fmt.Sprintf("%*s", maxLineSize, line)
	}

	// Combine the padded fileBase with the line number
	caller = fmt.Sprintf("%s:%s", fileBase, line)

	return term.Yellowf("[%s]", caller)
}

func formatTimestamp(i interface{}, timeLayout string) string {
	strTime, ok := i.(string)
	if !ok {
		return term.Cyanf("[%s]", i)
	}

	ts, err := time.ParseInLocation(time.RFC3339, strTime, time.Local)
	if err != nil {
		strTime = i.(string)
	} else {
		strTime = ts.In(time.Local).Format(timeLayout)
	}

	return term.Cyanf("[%s]", strTime)
}
