package logger

import (
	"context"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	defaultLevel  = "info"
	defaultFormat = "json"
)

func Configure(level, format, service string) {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.SetGlobalLevel(parseLevel(level))

	var output io.Writer = os.Stdout
	if strings.EqualFold(strings.TrimSpace(format), "console") {
		output = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	}

	base := zerolog.New(output).With().Timestamp()
	if service != "" {
		base = base.Str("service", service)
	}
	logger := base.Logger()
	log.Logger = logger
	zerolog.DefaultContextLogger = &log.Logger
}

func From(ctx context.Context) *zerolog.Logger {
	return zerolog.Ctx(ctx)
}

func With(ctx context.Context, logger zerolog.Logger) context.Context {
	return logger.WithContext(ctx)
}

func parseLevel(level string) zerolog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "":
		return zerolog.InfoLevel
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	case "disabled", "off":
		return zerolog.Disabled
	default:
		return zerolog.InfoLevel
	}
}

func DefaultLevel() string {
	return defaultLevel
}

func DefaultFormat() string {
	return defaultFormat
}
