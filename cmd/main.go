package main

import (
	"goscouter/internal/terminal"
)

func main() {
    restore, err := terminal.EnterRawMode()
    if err != nil {
        panic(err)
    }

    for {}

    defer restore()
}
