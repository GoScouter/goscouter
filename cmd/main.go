package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"

	"goscouter/internal/cmd"
	"goscouter/internal/logger"
	"goscouter/internal/terminal"
)

var interrupted atomic.Bool

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

    logger.Log.Info("Starting command manager")
    commandManager := cmd.NewCommandManager()

    sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        <-sigChan
        interrupted.Store(true)
    }()

    reader := bufio.NewReader(os.Stdin)
    for !interrupted.Load() {
        fmt.Print("> ")

        input, err := terminal.ReadLine(reader, os.Stdout)
        if err != nil {
            if errors.Is(err, terminal.ErrInterrupted) {
                // Ctrl-C: abandon the current line and prompt again.
                continue
            }
            if errors.Is(err, io.EOF) {
                // Ctrl-D on an empty line: exit the shell.
                break
            }
            break
        }

        input = strings.TrimSpace(input)
        parts := strings.Split(input, " ")

        command, err := commandManager.GetCommand(parts[0])
        if err != nil {
            fmt.Println(err)
            continue
        }

        err = command.Exec(parts[1:])
        if err != nil {
            if errors.Is(err, cmd.ErrExit) {
                break
            }

            fmt.Println(err)
            continue
        }
   }

    logger.Log.Info("Exiting terminal raw mode, restoring old state")
    defer restore()
}
