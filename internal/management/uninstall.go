package management

import (
	"context"
	"fmt"
	"goscouter/internal/module"
	"goscouter/internal/style"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GoScouter/sdk"
)

var namePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func UninstallModule(ctx context.Context, input string) error {
	author, name, ok := strings.Cut(strings.TrimSpace(input), ":")
	if !ok || !namePattern.MatchString(author) || !namePattern.MatchString(name) {
		return fmt.Errorf("invalid module, expected author:name (got %q)", input)
	}

	key := module.Key(sdk.ModuleNamespace{Author: author, Name: name})

	if module.IsInternalAuthor(author) {
		return fmt.Errorf("%s is a built-in module and cannot be uninstalled", key)
	}

	modulesDir, err := module.ExternalDir()
	if err != nil {
		return err
	}

	step("Looking for %s in %s", style.Bold(key), style.Dim(modulesDir))

	manager := module.NewManager()
	if err := manager.LoadExternals(ctx); err != nil {
		return err
	}

	if manager.GetInternal(key) != nil {
		return fmt.Errorf("%s is a built-in module and cannot be uninstalled", key)
	}

	external := manager.GetExternal(key)
	if external == nil {
		return fmt.Errorf("%s is not installed", key)
	}

	target, err := filepath.EvalSymlinks(external.Path)
	if err != nil {
		return err
	}
	root, err := filepath.EvalSymlinks(modulesDir)
	if err != nil {
		return err
	}
	if filepath.Dir(target) != root {
		return fmt.Errorf("refusing to remove %s: it lives outside %s", target, root)
	}

	if err := os.Remove(target); err != nil {
		return err
	}

	step("Uninstalled %s from %s", style.Bold(key), style.Dim(target))

	return nil
}
