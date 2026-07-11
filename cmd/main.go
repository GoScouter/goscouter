package main

import (
	"bufio"
	"errors"
	"flag"
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
	"goscouter/internal/module"
	"goscouter/internal/terminal"
	"goscouter/internal/web"
)

var BANNER = `
 ██████╗  ██████╗ ███████╗ ██████╗ ██████╗ ██╗   ██╗████████╗███████╗██████╗
██╔════╝ ██╔═══██╗██╔════╝██╔════╝██╔═══██╗██║   ██║╚══██╔══╝██╔════╝██╔══██╗
██║  ███╗██║   ██║███████╗██║     ██║   ██║██║   ██║   ██║   █████╗  ██████╔╝
██║   ██║██║   ██║╚════██║██║     ██║   ██║██║   ██║   ██║   ██╔══╝  ██╔══██╗
╚██████╔╝╚██████╔╝███████║╚██████╗╚██████╔╝╚██████╔╝   ██║   ███████╗██║  ██║
 ╚═════╝  ╚═════╝ ╚══════╝ ╚═════╝ ╚═════╝  ╚═════╝    ╚═╝   ╚══════╝╚═╝  ╚═╝
`


const (
	NAME = "GS"

	PURPLE = "\033[38;2;87;87;232m"
	RESET = "\033[0m"
)

var BUILD_TIME string

var interrupted atomic.Bool

func main() {
    targetSite := flag.String("target", "", "The site to target")
    flag.Parse()

    if *targetSite == "" {
        fmt.Println("Usage: gs --target <https://example.com>")
        os.Exit(1)
    }

    printBanner()

    err := logger.SetupLogger(logger.LoggerConfig{
        Console: false,
        Level:   slog.LevelInfo,
    })
    if err != nil {
        panic(err)
    }

    fmt.Printf("Targeting site: %s\n", *targetSite)
    if !web.IsValidSite(*targetSite) {
        fmt.Println("Cannot reach targeted website!")
        os.Exit(1)
    }

    fmt.Println("Successfully connected to the target website!")
    logger.Log.Info("Entering terminal raw mode")
    restore, err := terminal.EnterRawMode()
    if err != nil {
        panic(err)
    }

    logger.Log.Info("Loading modules")
    moduleManager := module.NewManager()

    logger.Log.Info("Starting command manager")
    commandManager := cmd.NewManager(*targetSite, moduleManager)

    sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        <-sigChan
        interrupted.Store(true)
    }()

    reader := bufio.NewReader(os.Stdin)
    for !interrupted.Load() {
        fmt.Print("(gs) > ")

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

        command, err := commandManager.Get(parts[0])
        if err != nil {
            fmt.Printf("%s\r\n", err)
            continue
        }

        err = command.Exec(parts[1:])
        if err != nil {
            if errors.Is(err, cmd.ErrExit) {
                break
            }

            fmt.Printf("%s\r\n", err)
            continue
        }
   }

    logger.Log.Info("Exiting terminal raw mode, restoring old state")
    defer restore()
}

func printBanner() {
	fmt.Print(PURPLE)
	fmt.Print(BANNER)
	fmt.Print(RESET)
	fmt.Println()

	buildTime := BUILD_TIME
	if buildTime == "" {
		buildTime = "unknown"
	}

	fmt.Printf("\t\t\t%s • %s\n\n", NAME, buildTime)
}
