package module

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"goscouter/internal/logger"
)

func Download(manifest *Manifest, version string) (string, error) {
    if manifest == nil {
        return "", fmt.Errorf("module manifest cannot be null")
    }

    cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	return DownloadTo(manifest, filepath.Join(cacheDir, "gs"), version)
}

func DownloadTo(manifest *Manifest, dir, version string) (string, error) {
    if manifest == nil {
        return "", fmt.Errorf("module manifest cannot be null")
    }

    fmt.Printf("Installing %s@%s\r\n", manifest.Name, version)
	logger.Log.Info(fmt.Sprintf("Resolving platform %q for module %s", runtime.GOOS, manifest.Name))

    key := version + "/" + runtime.GOOS + "-" + runtime.GOARCH
    release, ok := manifest.Releases[key]
	if !ok {
		return "", fmt.Errorf("cannot find matching release (%s)", key)
	}

	u, err := url.Parse(release.Binary)
	if err != nil {
		return "", err
	}

	name := path.Base(u.Path)
	if name == "" || name == "." || name == "/" {
		return "", fmt.Errorf("could not determine a binary name from %q", release.Binary)
	}

	if err = os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	binaryPath := filepath.Join(dir, name)

	if _, err = os.Stat(binaryPath); err == nil {
		return "", fmt.Errorf("module %q is already installed at %s", name, binaryPath)
	} else if !os.IsNotExist(err) {
		return "", err
	}

	fmt.Printf("Downloading %s\r\n", release.Binary)
	logger.Log.Info(fmt.Sprintf("Downloading binary %q to %s", release.Binary, binaryPath))

	resp, err := http.Get(release.Binary)
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
	if checksum != release.Checksum {
		f.Close()
		_ = os.Remove(binaryPath)
		return "", fmt.Errorf(
			"checksum mismatch: expected %s, got %s",
			release.Checksum,
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
