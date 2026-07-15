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

func LoadExternal() ([]sdk.Module, func(), error) {
	dir, err := externalDir()
	if err != nil {
		return nil, noopCleanup, err
	}

	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, noopCleanup, nil
	}
	if err != nil {
		return nil, noopCleanup, err
	}

	var mods []sdk.Module
	var bins []*sdk.Binary
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
		bins = append(bins, bin)
	}

	cleanup := func() {
		for _, b := range bins {
			_ = b.Close()
		}
	}
	return mods, cleanup, nil
}

func noopCleanup() {}
