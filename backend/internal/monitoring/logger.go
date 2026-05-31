package monitoring

import (
	"context"
	"log/slog"
	"os"
)

type Logger struct{ *slog.Logger }

func NewLogger() Logger { return Logger{slog.New(slog.NewJSONHandler(os.Stdout, nil))} }

func (l Logger) BookingEvent(ctx context.Context, event string, attrs ...any) {
	l.InfoContext(ctx, event, attrs...)
}
