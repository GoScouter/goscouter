//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || zos

package terminal

import (
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// makeRaw puts the terminal in raw mode for input only. term.MakeRaw also turns
// off output processing, which makes every "\n" printed afterwards move down
// without returning to column 0, so that is switched back on.
func makeRaw(fd, _ int) (restore func() error, err error) {
	old, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	restore = func() error { return term.Restore(fd, old) }

	termios, err := unix.IoctlGetTermios(fd, ioctlReadTermios)
	if err != nil {
		restore()
		return nil, err
	}
	termios.Oflag |= unix.OPOST | unix.ONLCR
	if err := unix.IoctlSetTermios(fd, ioctlWriteTermios, termios); err != nil {
		restore()
		return nil, err
	}

	return restore, nil
}
