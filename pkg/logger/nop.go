package logger

import (
	"io"
	"log/slog"
	"os"
)

// NewNop retorna um Logger que descarta todo output. Ideal para testes unitários.
func NewNop() *Logger {
	h := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	return &Logger{
		handler:     h,
		serviceName: "nop",
		level:       LevelDebug,
		exitFunc:    os.Exit,
	}
}
