package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"goscouter/internal/management"
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

	install := flag.String("install", "", "Module to install")
	uninstall := flag.String("uninstall", "", "Module to uninstall (author:name)")

	flag.Parse()

	if *install != "" {
		if err := management.InstallModule(context.Background(), *install); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to install modules %s. %v\n", *install, err)
			os.Exit(1)
			return
		}

		os.Exit(0)
		return
	}

	if *uninstall != "" {
		if err := management.UninstallModule(context.Background(), *uninstall); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to uninstall module %s. %v\n", *uninstall, err)
			os.Exit(1)
			return
		}

		os.Exit(0)
		return
	}

	if *version {
		fmt.Println("Version: " + versionString())
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
	if err := management.InstallModule(context.Background(), "https://github.com/GoScouter/nginx-module"); err != nil {
		panic(err)
	}

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
