package logger

import (
	"context"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"time"
)

func init() {
	// setup global logger
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

type Logger struct {
	zerolog.Logger
}
type Component interface {
	// LoggerComponent returns component name used in component loggers
	LoggerComponent() string
}

// New constructor
func New(verbose, pretty bool) Logger {
	logLevel := zerolog.InfoLevel
	if verbose {
		logLevel = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(logLevel)
	if pretty {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	}
	return Logger{log.Logger}
}

// Global returns current global logger
func Global() *Logger {
	return &Logger{log.Logger}
}

// Get returns context logger for component
func Get(ctx context.Context, c interface{}) Logger {
	return Ctx(ctx).Component(c)
}

// Ctx creates context logger
func Ctx(ctx context.Context) Logger {
	logger := zerolog.Ctx(ctx)
	return Logger{Logger: *logger}
}

// Component creates logger for specified component or returns current logger
func (l Logger) Component(c interface{}) Logger {
	if v, ok := c.(Component); ok {
		return l.WithComponent(v.LoggerComponent())
	}
	return l
}

// WithComponent creates child logger for named component
func (l Logger) WithComponent(name string) Logger {
	return Logger{Logger: l.With().Str("component", name).Logger()}
}
