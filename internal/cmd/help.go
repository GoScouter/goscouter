package cmd

import (
    "fmt"
)

type HelpCommand struct {
    Manager *Manager
}

func (cmd *HelpCommand) Name() string {
    return "help"
}

func (cmd *HelpCommand) Description() string {
    return "Returns list of avaiable built-in commands."
}

func (cmd *HelpCommand) Exec(args []string) error {
    msg := "List of avaiable built-in commands:\r\n"
    if cmd.Manager != nil {
        for _, command := range cmd.Manager.Commands {
            msg += fmt.Sprintf("[*] %s - %s\r\n", command.Name(), command.Description())
        }
    }

    msg += "\r\nList of available flags:\r\n"
    msg += "[*] --target -- Determines the site that goscouter will target (requires http/https prefix).\r\n"

    fmt.Printf("%s\r\n", msg)
    return nil
}
