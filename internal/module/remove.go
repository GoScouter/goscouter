package module

import (
	"fmt"
	"os"
	"path/filepath"

	"goscouter/internal/logger"
)

func Remove(name string) (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	return RemoveFrom(name, filepath.Join(cacheDir, "gs"))
}

func RemoveFrom(name, dir string) (string, error) {
	binaryPath := filepath.Join(dir, name)

	info, err := os.Stat(binaryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("module %q is not installed", name)
		}
		return "", err
	}

	if info.IsDir() {
		return "", fmt.Errorf("%q is not an installed module", name)
	}

	if err := os.Remove(binaryPath); err != nil {
		return "", err
	}

	fmt.Printf("Uninstalled %s from %s\r\n", name, binaryPath)
	logger.Log.Info(fmt.Sprintf("Removed module %s at %s", name, binaryPath))
	return binaryPath, nil
}
