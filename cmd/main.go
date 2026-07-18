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

	"goscouter/internal"
	"goscouter/internal/cmd"
	"goscouter/internal/logger"
	"goscouter/internal/module"
	"goscouter/internal/style"
	"goscouter/internal/terminal"
	"goscouter/internal/utils"
	"goscouter/internal/versions"
)

var (
	BUILD_TIME string
	VERSION    string
)

var interrupted atomic.Bool

func main() {
	version := flag.Bool("version", false, "Returns goscouter cli version")
    targetSite := flag.String("target", "", "The site to target")
    flag.Parse()

	if *version {
        fmt.Println("Version:", VERSION)
        os.Exit(0)
	}

	if *targetSite == "" {
		fmt.Println("Usage: gs --target <example.com>")
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

	if err = versions.SuggestUpdate(VERSION); err != nil {
		logger.Log.Warn("Update check failed", "error", err)
		fmt.Printf("%s\n\n", style.Error("Update: "+err.Error()))
	    return
    }

	fmt.Printf("%s %s\n\n", style.Gray("Target:"), style.Bold(*targetSite))
	logger.Log.Info("Entering terminal raw mode")
	state, err := terminal.NewShellState()
	if err != nil {
		panic(err)
	}

	logger.Log.Info("Loading modules")
	moduleManager := module.NewManager()

	logger.Log.Info("Starting command manager")
	commandManager, err := cmd.NewManager(*targetSite, moduleManager)
	if err != nil {
		panic(err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		interrupted.Store(true)
	}()

	reader := bufio.NewReader(os.Stdin)
	for !interrupted.Load() {
		fmt.Print(style.Prompt())

		input, err := terminal.ReadLine(reader, os.Stdout, state)
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
		if input == "" {
			// Blank line: just re-prompt instead of reporting an empty command.
			continue
		}

		state.AddHistory(input)
		parts := strings.Fields(input)

		command, err := commandManager.Get(parts[0])
		if err != nil {
			fmt.Printf("%s\r\n", style.Error(err.Error()))
			continue
		}

		err = command.Exec(parts[1:])
		if err != nil {
			if errors.Is(err, cmd.ErrExit) {
				break
			}

			fmt.Printf("%s\r\n", style.Error(err.Error()))
			continue
		}
	}

	logger.Log.Info("Exiting terminal raw mode, restoring old state")
	defer state.Restore()
}

func printBanner() {
	buildTime := BUILD_TIME
	if buildTime == "" {
		buildTime = "unknown"
	}
	internal.BuildTime = buildTime

	version := VERSION
	if version == "" {
		version = "dev"
	}
	internal.Version = version

    utils.PrintBanner(internal.Version, internal.BuildTime)
}
