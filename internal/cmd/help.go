package cmd

import (
    "fmt"
)

type HelpCommand struct {
    Commands []Command
}

func (cmd *HelpCommand) Name() string {
    return "help"
}

func (cmd *HelpCommand) Description() string {
    return "Returns list of avaiable built-in commands."
}

func (cmd *HelpCommand) Exec(args []string) error {
    msg := "List of avaiable built-in commands:\r\n"
    msg += fmt.Sprintf("[*] %s - %s\r\n", cmd.Name(), cmd.Description())
    for _, command := range cmd.Commands {
        msg += fmt.Sprintf("[*] %s - %s\r\n", command.Name(), command.Description())
    }

    msg += "\r\nList of available flags:\r\n"
    msg += "[*] --target -- Determines the site that goscouter will target.\r\n"

    fmt.Printf("%s\r\n", msg)
    return nil
}
