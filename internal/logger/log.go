package logger

import (
	"io"
	"log/slog"
	"path/filepath"
    "os"
)

type LoggerConfig struct {
	Console bool
	Level   slog.Level
}

var Log *slog.Logger

func LogPath() (string, error) {
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dir = filepath.Join(dir, "goscouter")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(dir, "goscouter.log"), nil
}

func SetupLogger(cfg LoggerConfig) error {
    logPath, err := LogPath()
    if err != nil {
        return err
    }

    file, err := os.OpenFile(
		logPath,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)
	if err != nil {
		return err
	}

	var writer io.Writer = file
	if cfg.Console {
		writer = io.MultiWriter(os.Stdout, file)
	}

	opts := &slog.HandlerOptions{
		Level:     cfg.Level,
		AddSource: true,
	}

	handler := slog.NewTextHandler(writer, opts)
    Log = slog.New(handler)

    return nil
}
