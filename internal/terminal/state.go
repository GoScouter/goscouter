package terminal

import (
	"os"
)

const (
	// Switch to the alternate screen and clear it, like vim does. Leaving it
	// brings back whatever the terminal showed before.
	enterAltScreen = "\x1b[?1049h\x1b[H\x1b[2J"
	leaveAltScreen = "\x1b[?1049l"
)

type ShellState struct {
	Fd             int
	CommandHistory []string
	Screen         *Screen

	historyPath string
	restore     func() error
}

// The Screen captures os.Stdout and os.Stderr from here on, until Restore.
func NewShellState(historyPath string) (*ShellState, error) {
	history, err := loadHistory(historyPath)
	if err != nil {
		history = nil
	}
	if len(history) > maxHistory {
		history = history[len(history)-maxHistory:]
		_ = saveHistory(historyPath, history)
	}

	fd := int(os.Stdin.Fd())
	restore, err := makeRaw(fd, int(os.Stdout.Fd()))
	if err != nil {
		return nil, err
	}

	screen, err := startScreen(os.Stdin, os.Stdout)
	if err != nil {
		restore()
		return nil, err
	}

	return &ShellState{
		Fd:             fd,
		CommandHistory: history,
		Screen:         screen,
		historyPath:    historyPath,
		restore:        restore,
	}, nil
}

func (state *ShellState) Restore() error {
	clear(state.CommandHistory)
	state.Screen.Close()
	return state.restore()
}

func (state *ShellState) AddHistory(cmd string) {
	if cmd == "" {
		return
	}

	if n := len(state.CommandHistory); n > 0 && state.CommandHistory[n-1] == cmd {
		return
	}

	state.CommandHistory = append(state.CommandHistory, cmd)
	if state.historyPath != "" {
		_ = appendHistory(state.historyPath, cmd)
	}
}
