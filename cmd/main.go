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
	"goscouter/internal/logging"
	"goscouter/internal/module"
	"goscouter/internal/terminal"
	"goscouter/internal/utils"
	"goscouter/internal/versions"

	"github.com/GoScouter/sdk/style"
)

var (
	BUILD_TIME string
	VERSION    string
)

var interrupted atomic.Bool

func main() {
	internal.Version = versionString()

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
	// The update check can prompt and exit, so it runs on the normal screen
	// before the shell takes over the terminal.
	if err := versions.SuggestUpdate(VERSION); err != nil {
		return fmt.Errorf("update check: %w", err)
	}

	historyPath, _ := terminal.HistoryPath()

	state, err := terminal.NewShellState(historyPath)
	if err != nil {
		return err
	}
	defer state.Restore()

	// The logger kept the original stdout, so point it at the screen too.
	logging.SetOutput(os.Stdout)

	printBanner()
	fmt.Printf("%s %s\n\n", style.Gray("Target:"), style.Bold(target))

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
	commandManager.History = func() []string { return state.CommandHistory }
	state.Screen.SetTarget(target)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		interrupted.Store(true)
		// The prompt is waiting for a key; this wakes it so the shell can exit
		// and give the terminal back.
		state.Screen.CloseInput()
	}()

	runner.CleanupState()

	reader := bufio.NewReader(state.Screen.Input())
	for !interrupted.Load() {
		input, err := terminal.ReadLine(reader, os.Stdout, state, terminal.LineOptions{
			Prompt:   style.Prompt(),
			Complete: commandManager.Complete,
			Hint:     commandManager.Hint,
		})
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

		// Marks the prompt line above as the start of this command's output,
		// so it can be folded.
		fmt.Print(terminal.BlockMarker)

		if line, ok := strings.CutPrefix(input, cmd.ShellPrefix); ok {
			if err := runShell(state, strings.TrimSpace(line)); err != nil {
				fmt.Printf("%s\r\n", style.Gray("✗ process failed: "+err.Error()))
			}
			continue
		}

		parts := strings.Fields(input)

		command, err := commandManager.Get(parts[0])
		if err != nil {
			fmt.Printf("%s\r\n", style.Error(err.Error()))
			continue
		}

		err = execute(state, command, parts[1:])
		state.Screen.SetTarget(commandManager.Target)
		if err != nil {
			if errors.Is(err, cmd.ErrExit) {
				break
			}
			if errors.Is(err, context.Canceled) {
				fmt.Printf("%s\r\n", style.Gray("✗ cancelled"))
				continue
			}

			fmt.Printf("%s\r\n", style.Error(err.Error()))
			continue
		}
		runner.CleanupState()
	}

	return nil
}

func execute(state *terminal.ShellState, command cmd.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stop := state.Screen.OnInterrupt(cancel)
	defer stop()

	state.Screen.Busy(command.Name())
	defer state.Screen.Idle()

	err := command.Exec(ctx, args)
	if ctx.Err() != nil {
		return context.Canceled
	}
	return err
}

func runShell(state *terminal.ShellState, line string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stop := state.Screen.OnInterrupt(cancel)
	defer stop()

	state.Screen.Busy("!" + line)
	defer state.Screen.Idle()

	return cmd.RunShell(ctx, line)
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
