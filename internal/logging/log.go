package logging

import (
	"fmt"
	"io"
	"os"
	"sync"

	"goscouter/internal/style"
)

var (
	mu      sync.Mutex
	out     io.Writer = os.Stdout
	enabled           = true
)

func SetOutput(w io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	out = w
}

func SetEnabled(on bool) {
	mu.Lock()
	defer mu.Unlock()
	enabled = on
}

func line(s string) {
	mu.Lock()
	defer mu.Unlock()

	if !enabled {
		return
	}
	fmt.Fprint(out, s+"\r\n")
}

func Found(format string, a ...any) { line(style.Foundf(format, a...)) }

func Failed(format string, a ...any) { line(style.Failuref(format, a...)) }

func Alert(format string, a ...any) { line(style.Alertf(format, a...)) }

func Step(format string, a ...any) { line(style.Infof(format, a...)) }
