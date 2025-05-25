package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Logger struct {
	Level zerolog.Level
}

type LoggerInterface interface {
	GetLogger() zerolog.Logger
}

func NewLogger(level string) *Logger {
	setup(level)

	return &Logger{
		Level: getLevel(level),
	}
}

func setup(level string) {
	zerolog.SetGlobalLevel(getLevel(level))

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	})
}

func (l *Logger) GetLogger() zerolog.Logger {
	return log.Logger
}

func getLevel(level string) zerolog.Level {
	switch level {
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "debug":
		return zerolog.DebugLevel
	case "trace":
		return zerolog.TraceLevel
	default:
		return zerolog.InfoLevel
	}
}
