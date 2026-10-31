//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris && !zos && !windows

package terminal

import "golang.org/x/term"

func makeRaw(fd, _ int) (restore func() error, err error) {
	old, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	return func() error { return term.Restore(fd, old) }, nil
}
