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

// Configure executes the logger.Configure operation.
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

// From executes the logger.From operation.
func From(ctx context.Context) *zerolog.Logger {
	return zerolog.Ctx(ctx)
}

// TraceCall records a function entry on the logger carried by the context.
func TraceCall(ctx context.Context, function string) {
	From(ctx).Debug().Str("function", function).Msg("function called")
}

// TraceErr logs a returning error with caller, then returns err unchanged.
func TraceErr(ctx context.Context, msg string, err error) error {
	if err == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	From(ctx).Error().Err(err).Caller(1).Msg(msg)
	return err
}

// With executes the logger.With operation.
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

// DefaultLevel executes the logger.DefaultLevel operation.
func DefaultLevel() string {
	return defaultLevel
}

// DefaultFormat executes the logger.DefaultFormat operation.
func DefaultFormat() string {
	return defaultFormat
}
