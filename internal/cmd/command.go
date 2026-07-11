package cmd

import "fmt"

var registry = map[string]Command{}

type Command interface {
    Name() string
    Exec(args []string) error
}

type CommandManager struct {
    Commands map[string]Command
}

func NewCommandManager() *CommandManager {
    cm := &CommandManager {
        Commands: make(map[string]Command),
    }

    cm.AddCommand(&ExitCommand{})
    return cm
}

func (cm *CommandManager) GetCommand(name string) (Command, error) {
    cmd, ok := cm.Commands[name]
    if !ok {
        return nil, fmt.Errorf("%s - command does not exists", name)
    }

    return cmd, nil
}

func (cm *CommandManager) AddCommand(cmd Command) {
    cm.Commands[cmd.Name()] = cmd
}

func (cm *CommandManager) RemoveCommand(name string) {
    delete(cm.Commands, name)
}
