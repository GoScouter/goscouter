package cmd

import (
	"fmt"
	"goscouter/internal/logger"
	"goscouter/internal/module"
	"maps"
	"slices"
)

type Command interface {
    Name() string
    Description() string
    Exec(args []string) error
}

type Manager struct {
    Commands map[string]Command
}

func NewManager(target string, modules *module.Manager) *Manager {
    cm := &Manager {
        Commands: make(map[string]Command),
    }

    logger.Log.Info("Loading built-in commands")
    cm.Add(&ExitCommand{})
    cm.Add(&ClearCommand{})

    if modules != nil {
        if recordsMod, err := modules.Get("records"); err == nil {
            cm.Add(&RecordsCommand{Target: target, Module: recordsMod})
        } else {
            logger.Log.Error(err.Error())
        }
    }

    cm.Add(&HelpCommand{
        Commands: slices.Collect(maps.Values(cm.Commands)),
    })

    logger.Log.Info("Loaded built-in commands.")
    for k := range cm.Commands {
        logger.Log.Info(fmt.Sprintf("%s command", k))
    }

    return cm
}

func (cm *Manager) Get(name string) (Command, error) {
    cmd, ok := cm.Commands[name]
    if !ok {
        return nil, fmt.Errorf("%s - command does not exists", name)
    }

    return cmd, nil
}

func (cm *Manager) Add(cmd Command) {
    cm.Commands[cmd.Name()] = cmd
}

func (cm *Manager) Remove(name string) {
    delete(cm.Commands, name)
}
