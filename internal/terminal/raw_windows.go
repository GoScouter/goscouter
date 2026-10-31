//go:build windows

package terminal

import (
	"golang.org/x/sys/windows"
	"golang.org/x/term"
)

// makeRaw puts the console input in raw mode without Quick Edit, which would
// otherwise swallow mouse events, and makes sure the output understands ANSI
// escape sequences and returns to column 0 on "\n".
func makeRaw(fd, outFd int) (restore func() error, err error) {
	old, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}

	in := windows.Handle(fd)
	var inMode uint32
	if err := windows.GetConsoleMode(in, &inMode); err == nil {
		windows.SetConsoleMode(in, (inMode|windows.ENABLE_EXTENDED_FLAGS)&^windows.ENABLE_QUICK_EDIT_MODE)
	}

	out := windows.Handle(outFd)
	var outMode uint32
	if err := windows.GetConsoleMode(out, &outMode); err != nil {
		// Output is redirected, so there is no console mode to adjust.
		return func() error { return term.Restore(fd, old) }, nil
	}

	raw := outMode | windows.ENABLE_PROCESSED_OUTPUT | windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
	raw &^= windows.DISABLE_NEWLINE_AUTO_RETURN
	if err := windows.SetConsoleMode(out, raw); err != nil {
		term.Restore(fd, old)
		return nil, err
	}

	return func() error {
		windows.SetConsoleMode(out, outMode)
		return term.Restore(fd, old)
	}, nil
}
