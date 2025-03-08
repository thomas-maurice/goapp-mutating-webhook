package log

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/charmbracelet/log"
)

func GetLogger() *slog.Logger {
	formatter := log.TextFormatter
	if os.Getenv("LOG_FORMAT") == "json" {
		formatter = log.JSONFormatter
	}
	l := log.NewWithOptions(os.Stderr, log.Options{
		TimeFormat:      time.RFC3339,
		ReportTimestamp: true,
		ReportCaller:    true,
		Formatter:       formatter,
	})
	return slog.New(l)
}

func LogFromContext(ctx context.Context) *slog.Logger {
	lg, ok := ctx.Value("logger").(*slog.Logger)
	if ok {
		return lg
	}
	return GetLogger()
}
