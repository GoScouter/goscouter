package terminal

import (
	"os"

	"golang.org/x/term"
)

func EnterRawMode() (restore func() error, err error) {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}

	return func() error { return term.Restore(fd, oldState) }, nil
}
