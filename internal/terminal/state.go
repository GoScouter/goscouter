package terminal

import (
	"os"

	"golang.org/x/term"
)

type ShellState struct {
    Fd int
    OldState *term.State
    CommandHistory []string
    HistoryIndex int
}

func NewShellState() (*ShellState, error) {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}

    return &ShellState{
        Fd: fd,
        OldState: oldState,
        CommandHistory: make([]string, 0),
    }, nil
}

func (state *ShellState) Restore() error {
    state.HistoryIndex = 0
    clear(state.CommandHistory)
    return term.Restore(state.Fd, state.OldState)
}

func (state *ShellState) AddHistory(cmd string) {
    if cmd == "" {
        return
    }

    state.CommandHistory = append(state.CommandHistory, cmd)
}

func (state *ShellState) Move(diraction rune) string {
    historyLen := len(state.CommandHistory)
    if historyLen == 0 {
        return ""
    }

    if diraction == Up {
        state.HistoryIndex += 1
        if state.HistoryIndex >= historyLen {
            state.HistoryIndex = historyLen - 1
        }
    } else if diraction == Down {
        state.HistoryIndex -= 1
        if state.HistoryIndex < 0 {
            state.HistoryIndex = 0
        }
    }

    return state.CommandHistory[state.HistoryIndex]
}

