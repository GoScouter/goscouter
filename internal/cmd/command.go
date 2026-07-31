package cmd

import (
	"fmt"
	"goscouter/internal/module"
	"maps"
	"regexp"
	"slices"
	"strings"

	"goscouter/internal/logger"
)

type Command interface {
	Name() string
	Description() string
	Exec(args []string) error
}

type CommandManager struct {
	Commands map[string]Command
	Target   string
}

func (cm *CommandManager) SetTarget(target string) {
	cm.Target = target
}

func NewManager(target string, manager *module.Manager) (*CommandManager, error) {
	cm := &CommandManager{
		Commands: make(map[string]Command),
		Target:   target,
	}

	logger.Log.Info("Loading built-in commands")
	cm.addCommand(&InfoCommand{})
	cm.addCommand(&ExitCommand{})
	cm.addCommand(&ClearCommand{})
	cm.addCommand(&HelpCommand{Manager: cm})
	cm.addCommand(&TargetCommand{Manager: cm})

	logger.Log.Info("Loaded built-in commands.")

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

func filter[T any](s []T, predicate func(T) bool) []T {
	result := make([]T, 0, len(s)) // Pre-allocate for efficiency
	for _, v := range s {
		if predicate(v) {
			result = append(result, v)
		}
	}
	return result
}

var namespaceRegex = regexp.MustCompile(`^[A-Za-z]+:[A-Za-z]+$`)

func (cm *CommandManager) Get(name string) (Command, error) {
	if matches := len(namespaceRegex.FindStringSubmatch(name)); matches > 0 {
		cmd, ok := cm.Commands[name]
		if ok {
			return cmd, nil
		}
	}

	keys := slices.Sorted(maps.Keys(cm.Commands))
	suffixKeys := filter(keys, func(v string) bool {
		return strings.HasSuffix(v, name)
	})

	if len(suffixKeys) > 0 {
		return cm.Commands[suffixKeys[0]], nil
	}

	return nil, fmt.Errorf("%s - command does not exists", name)
}

func (cm *CommandManager) addCommand(cmd Command) {
	cm.Commands[cmd.Name()] = cmd
}

func (cm *CommandManager) removeCommand(name string) {
	delete(cm.Commands, name)
}
