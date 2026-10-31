package cmd

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"goscouter/internal/module"

	"github.com/GoScouter/sdk/style"
)

type Command interface {
	Name() string
	Description() string
	Exec(ctx context.Context, args []string) error
}

type CommandManager struct {
	Commands map[string]Command
	Target   string

	Last *Result
	History func() []string
}

func (cm *CommandManager) SetTarget(target string) {
	cm.Target = target
}

func NewManager(target string, manager *module.Manager) (*CommandManager, error) {
	cm := &CommandManager{
		Commands: make(map[string]Command),
		Target:   target,
	}

	cm.addCommand(&InfoCommand{})
	cm.addCommand(&ExitCommand{})
	cm.addCommand(&ClearCommand{})
	cm.addCommand(&HelpCommand{Manager: cm})
	cm.addCommand(&TargetCommand{Manager: cm})
	cm.addCommand(&SaveCommand{Manager: cm})

	if manager == nil {
		return cm, nil
	}

	reporter, err := module.DialReporter()
	if err != nil {
		return nil, err
	}

	for _, internal := range manager.GetInternals() {
		cm.addCommand(&InternalCommand{
			CmdManager:    cm,
			ModuleManager: manager,
			Module:        internal,
			Reporter:      reporter,
		})
	}

	for _, external := range manager.GetExternals() {
		cm.addCommand(&ExternalCommand{
			CmdManager:    cm,
			ModuleManager: manager,
			External:      external,
			Reporter:      reporter,
		})
	}

	return cm, nil
}

func (cm *CommandManager) Get(name string) (Command, error) {
	if cmd, ok := cm.Commands[name]; ok {
		return cmd, nil
	}

	var matches []string
	for key := range cm.Commands {
		if shortName(key) == name {
			matches = append(matches, key)
		}
	}

	switch len(matches) {
	case 1:
		return cm.Commands[matches[0]], nil
	case 0:
	default:
		slices.Sort(matches)
		return nil, fmt.Errorf("%s is ambiguous, use one of: %s",
			style.Bold(name), strings.Join(matches, ", "))
	}

	return nil, fmt.Errorf("unknown command %s\n  %s %s %s",
		style.Bold(name),
		style.Gray("type"), style.Bold(style.Cyan("help")), style.Gray("to see the available commands"))
}

func (cm *CommandManager) addCommand(cmd Command) {
	cm.Commands[cmd.Name()] = cmd
}

func (cm *CommandManager) removeCommand(name string) {
	delete(cm.Commands, name)
}
