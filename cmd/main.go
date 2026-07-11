package main

import (
    "log/slog"

	"goscouter/internal/logger"
	"goscouter/internal/terminal"
)

func main() {
    err := logger.SetupLogger(logger.LoggerConfig{
        Console: false,
        Level:   slog.LevelInfo,
    })
    if err != nil {
        panic(err)
    }

    logger.Log.Info("Entering terminal raw mode")
    restore, err := terminal.EnterRawMode()
    if err != nil {
        panic(err)
    }

    // for {}

    logger.Log.Info("Exiting terminal raw mode, restoring old state")
    defer restore()
}
