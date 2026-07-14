package module

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"fmt"

	"goscouter/internal/logger"
)

func Download(manifest Manifest) (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	return DownloadTo(manifest, filepath.Join(cacheDir, "gs"))
}

func DownloadTo(manifest Manifest, dir string) (string, error) {
	fmt.Printf("Installing %s@%s\r\n", manifest.Name, manifest.Version)
	logger.Log.Info(fmt.Sprintf("Resolving platform %q for module %s", runtime.GOOS, manifest.Name))

	platform, ok := manifest.Platforms[runtime.GOOS]
	if !ok {
		return "", fmt.Errorf("cannot find binary matching your platform (%s)", runtime.GOOS)
	}

	u, err := url.Parse(platform.Binary)
	if err != nil {
		return "", err
	}

	name := path.Base(u.Path)
	if name == "" || name == "." || name == "/" {
		return "", fmt.Errorf("could not determine a binary name from %q", platform.Binary)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	binaryPath := filepath.Join(dir, name)

	if _, err := os.Stat(binaryPath); err == nil {
		return "", fmt.Errorf("module %q is already installed at %s", name, binaryPath)
	} else if !os.IsNotExist(err) {
		return "", err
	}

	fmt.Printf("Downloading %s\r\n", platform.Binary)
	logger.Log.Info(fmt.Sprintf("Downloading binary %q to %s", platform.Binary, binaryPath))

	resp, err := http.Get(platform.Binary)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("no binary was found (HTTP %d)", resp.StatusCode)
	}

	f, err := os.OpenFile(binaryPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return "", err
	}

	hasher := sha256.New()
	writer := io.MultiWriter(f, hasher)

	written, err := io.Copy(writer, resp.Body)
	if err != nil {
		// Windows cannot remove a file while a handle is open, so close first.
		f.Close()
		_ = os.Remove(binaryPath)
		return "", err
	}

	checksum := hex.EncodeToString(hasher.Sum(nil))
	if checksum != platform.Checksum {
		f.Close()
		_ = os.Remove(binaryPath)
		return "", fmt.Errorf(
			"checksum mismatch: expected %s, got %s",
			platform.Checksum,
			checksum,
		)
	}

	logger.Log.Info(fmt.Sprintf("Verified checksum %s (%d bytes)", checksum, written))
	if err := f.Chmod(0o755); err != nil {
		f.Close()
		_ = os.Remove(binaryPath)
		return "", fmt.Errorf("failed to make binary executable: %w", err)
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(binaryPath)
		return "", err
	}

	fmt.Printf("Installed %s (%d bytes) to %s\r\n", manifest.Name, written, binaryPath)
	logger.Log.Info(fmt.Sprintf("Installed module %s to %s", manifest.Name, binaryPath))
	return binaryPath, nil
}
