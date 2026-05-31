package monitoring

import (
	"log/slog"
	"os"
)

func NewLogger(service string) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: true})).With("service", service)
}
