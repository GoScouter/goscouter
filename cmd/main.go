package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"

	"goscouter/internal"
	"goscouter/internal/cmd"
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
		fmt.Println(versionString())
		return
	}

	if *targetSite == "" {
		fmt.Fprintln(os.Stderr, "Usage: gs --target <example.com>")
		os.Exit(1)
	}

	if err := run(*targetSite); err != nil {
		fmt.Fprintf(os.Stderr, "%s\r\n", style.Error(err.Error()))
		os.Exit(1)
	}
}

func run(target string) error {
	printBanner()

	if err := versions.SuggestUpdate(VERSION); err != nil {
		return fmt.Errorf("update check: %w", err)
	}

	fmt.Printf("%s %s\n\n", style.Gray("Target:"), style.Bold(target))

	state, err := terminal.NewShellState()
	if err != nil {
		return err
	}
	defer state.Restore()

	moduleManager := module.NewManager()
	if err := moduleManager.LoadExternals(context.Background()); err != nil {
		return err
	}

	// An incomplete graph only disables the modules that depend on what is
	// missing, so report it and keep going.
	if _, err := moduleManager.Build(); err != nil {
		fmt.Printf("%s\r\n", style.Alertf("modules: %v", err))
	}

	runner, err := module.CreateRunner()
	if err != nil {
		return err
	}

	go func() {
		if err := runner.Start(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "%s\r\n", style.Errorf("runner: %v", err))
		}
	}()

	commandManager, err := cmd.NewManager(target, moduleManager)
	if err != nil {
		return err
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
		runner.CleanupState()
	}

	return nil
}

// versionString falls back to "dev" for builds made without the release ldflags.
func versionString() string {
	if VERSION == "" {
		return "dev"
	}
	return VERSION
}

func printBanner() {
	buildTime := BUILD_TIME
	if buildTime == "" {
		buildTime = "unknown"
	}
	internal.BuildTime = buildTime
	internal.Version = versionString()

	utils.PrintBanner(internal.Version, internal.BuildTime)
}
