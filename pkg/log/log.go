package log

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/charmbracelet/log"
)

type LoggerKey struct{}

var (
	loggerKey LoggerKey = LoggerKey{}
)

func GetLogger() *slog.Logger {
	formatter := log.TextFormatter
	if os.Getenv("LOG_FORMAT") == "json" {
		formatter = log.JSONFormatter
	}

	lvl := log.InfoLevel
	if os.Getenv("DEBUG") != "" {
		lvl = log.DebugLevel
	}

	l := log.NewWithOptions(os.Stderr, log.Options{
		TimeFormat:      time.RFC3339,
		ReportTimestamp: true,
		ReportCaller:    true,
		Formatter:       formatter,
		Level:           lvl,
	})

	return slog.New(l)
}

func FromContext(ctx context.Context) *slog.Logger {
	lg, ok := ctx.Value(loggerKey).(*slog.Logger)
	if ok {
		return lg
	}

	return GetLogger()
}

func ToContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}
