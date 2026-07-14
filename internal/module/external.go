package module

import (
	"fmt"
	"os"
	"path/filepath"

	"goscouter/internal/logger"

	"github.com/GoScouter/sdk"
)

func externalDir() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "gs"), nil
}

func LoadExternal() ([]sdk.Module, error) {
	dir, err := externalDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var mods []sdk.Module
	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		path := filepath.Join(dir, e.Name())
		bin, err := sdk.Open(path)
		if err != nil {
			logger.Log.Warn(fmt.Sprintf("skipping external module %q: %v", path, err))
			continue
		}
		mods = append(mods, bin)
	}

	return mods, nil
}
