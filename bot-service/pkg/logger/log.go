package logger

import (
	"log"
	"log/slog"
	"os"
)

var Log *slog.Logger

func Init(level string) {
	var slogLevel slog.Level
	switch level {
	case "debug":
		slogLevel = slog.LevelDebug
	case "info":
		slogLevel = slog.LevelInfo
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: slogLevel,
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	Log = slog.New(handler)
	slog.SetDefault(Log)

	log.Printf("Логгер инициализирован с уровнем: %s", level)
}
