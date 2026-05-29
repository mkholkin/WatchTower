package configs

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// NewLoggerFromConfig builds slog logger based on logging section.
func NewLoggerFromConfig(cfg LoggingConfig) (*slog.Logger, io.Closer, error) {
	level := parseLogLevel(cfg.Level)

	writer, closer, err := resolveLogWriter(cfg.Output)
	if err != nil {
		return nil, nil, err
	}

	handler := resolveLogHandler(cfg.Format, writer, level)
	return slog.New(handler), closer, nil
}

func parseLogLevel(raw string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "", "info":
		return slog.LevelInfo
	default:
		_, _ = fmt.Fprintf(os.Stderr, "unknown log level %q, fallback to info\n", raw)
		return slog.LevelInfo
	}
}

func resolveLogWriter(raw string) (io.Writer, io.Closer, error) {
	output := strings.ToLower(strings.TrimSpace(raw))
	switch output {
	case "", "stdout":
		return os.Stdout, nil, nil
	case "stderr":
		return os.Stderr, nil, nil
	default:
		f, err := os.OpenFile(output, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, nil, err
		}
		return f, f, nil
	}
}

func resolveLogHandler(format string, writer io.Writer, level slog.Level) slog.Handler {
	opts := &slog.HandlerOptions{AddSource: true, Level: level}

	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		return slog.NewJSONHandler(writer, opts)
	case "", "text":
		return slog.NewTextHandler(writer, opts)
	default:
		_, _ = fmt.Fprintf(os.Stderr, "unknown log format %q, fallback to text\n", format)
		return slog.NewTextHandler(writer, opts)
	}
}
