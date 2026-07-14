package cmd

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"goscouter/internal/logger"
	"goscouter/internal/module"
)

func execSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func commandName(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, execSuffix())
}

type Command interface {
    Name() string
    Description() string
    Exec(args []string) error
}

type Manager struct {
    Commands map[string]Command
}

func filePathWalkDir(root string) ([]string, error) {
    var files []string
    err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
        if !info.IsDir() {
            files = append(files, path)
        }
        return nil
    })
    return files, err
}

func NewManager(target string, moduleManager *module.Manager) (*Manager, error) {
    cm := &Manager {
        Commands: make(map[string]Command),
    }

    logger.Log.Info("Loading built-in commands")
    cm.Add(&ExitCommand{})
    cm.Add(&ClearCommand{})
    cm.Add(&InstallCommand{})

    cm.Add(&HelpCommand{
        Commands: slices.Collect(maps.Values(cm.Commands)),
    })

    logger.Log.Info("Loaded built-in commands.")
    for k := range cm.Commands {
        logger.Log.Info(fmt.Sprintf("%s command", k))
    }

    logger.Log.Info("Loading external commands")
    cacheDir, err := os.UserCacheDir()
    if err != nil {
        return nil, err
    }

    cacheDir = filepath.Join(cacheDir, "gs")
    if err := os.MkdirAll(cacheDir, 0o755); err != nil {
        return nil, err
    }

    if moduleManager != nil {
		mods := moduleManager.GetAll()
		for _, mod := range mods {
			cm.Add(&ModuleCommand{
				Target: target,
				Module: mod,
			})
		}
    }

    logger.Log.Info(fmt.Sprintf("Looking at: %s", cacheDir))
    external, err := filePathWalkDir(cacheDir)
    if err != nil {
        return nil, err
    }

    for _, ex := range external {
		cm.Add(&ExternalCommand{
            Target: target,
            ModuleName: commandName(ex),
            Module: ex,
        })
    }

    return cm, nil
}

func (cm *Manager) Get(name string) (Command, error) {
    cmd, ok := cm.Commands[name]
    if ok {
        return cmd, nil
    }

    return nil, fmt.Errorf("%s - command does not exists", name)
}

func (cm *Manager) Add(cmd Command) {
    cm.Commands[cmd.Name()] = cmd
}

func (cm *Manager) Remove(name string) {
    delete(cm.Commands, name)
}
