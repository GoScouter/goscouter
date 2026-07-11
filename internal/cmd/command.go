package cmd

import (
	"fmt"
	"goscouter/internal/logger"
	"maps"
	"slices"
)

type Command interface {
    Name() string
    Description() string
    Exec(args []string) error
}

type CommandManager struct {
    Commands map[string]Command
}

func NewCommandManager() *CommandManager {
    cm := &CommandManager {
        Commands: make(map[string]Command),
    }

    logger.Log.Info("Loading built-in commands")
    cm.AddCommand(&ExitCommand{})
    cm.AddCommand(&ClearCommand{})
    cm.AddCommand(&HelpCommand{
        Commands: slices.Collect(maps.Values(cm.Commands)),
    })

    logger.Log.Info("Loaded built-in commands.")
    for k := range cm.Commands {
        logger.Log.Info(fmt.Sprintf("%s command", k))
    }

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
