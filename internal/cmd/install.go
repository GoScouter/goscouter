package cmd

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
	"goscouter/internal/module"
)

type InstallCommand struct {
	Manager *Manager
	Target  string
}

func (cmd *InstallCommand) Name() string {
	return "install"
}

func (cmd *InstallCommand) Description() string {
	return "Installs a module"
}

func (cmd *InstallCommand) Exec(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: install <module-ref>")
	}

	link := args[0]
	if link == "" {
		return fmt.Errorf("usage: install <module-ref>")
	}

	logger.Log.Info(fmt.Sprintf("Installing module from %q\r\n", link))

	ref := module.ParseModule(link)
	if ref == nil {
		fmt.Printf("Resolving manifest from %s\r\n", link)
		manifest, err := module.ResolveManifest(link)
		if err != nil {
			return err
		}

		binaryPath, err := download(manifest)
		if err != nil {
			return err
		}

		return cmd.register(binaryPath)
	}

	fmt.Printf("Resolving module %s\r\n", ref.ToString())

	// Make sure to make the registry website to have an api endpoint that doing this.
	// Needs a domain for that because github pages cannot do this.
	const rawBase = "https://raw.githubusercontent.com/GoScouter/registry/main"
	url := fmt.Sprintf("%s/%s/%s/%s/manifest.json", rawBase, ref.Author, ref.Module, ref.Version)

	logger.Log.Info(fmt.Sprintf("Fetching manifest from %s", url))
	manifest, err := module.ResolveManifest(url)
	if err != nil {
		return err
	}

	if ref.Module != manifest.Name {
		return fmt.Errorf("module name mismatch (%s != %s)", ref.Module, manifest.Name)
	}

	if ref.Version != manifest.Version {
		return fmt.Errorf("module version mismatch (%s != %s)", ref.Version, manifest.Version)
	}

	binaryPath, err := download(manifest)
	if err != nil {
		return err
	}

	return cmd.register(binaryPath)
}

func (cmd *InstallCommand) register(binaryPath string) error {
	if cmd.Manager == nil {
		return nil
	}

	name := commandName(binaryPath)
	cmd.Manager.Add(&ExternalCommand{
		Target:     cmd.Target,
		ModuleName: name,
		Module:     binaryPath,
	})

	fmt.Printf("Command %q is now available\r\n", name)
	logger.Log.Info(fmt.Sprintf("Registered external command %q from %s", name, binaryPath))
	return nil
}

func download(manifest module.Manifest) (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	return downloadTo(manifest, filepath.Join(cacheDir, "gs"))
}

func downloadTo(manifest module.Manifest, dir string) (string, error) {
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
	defer f.Close()

	hasher := sha256.New()
	writer := io.MultiWriter(f, hasher)

	written, err := io.Copy(writer, resp.Body)
	if err != nil {
		_ = os.Remove(binaryPath)
		return "", err
	}

	checksum := hex.EncodeToString(hasher.Sum(nil))
	if checksum != platform.Checksum {
		_ = os.Remove(binaryPath)
		return "", fmt.Errorf(
			"checksum mismatch: expected %s, got %s",
			platform.Checksum,
			checksum,
		)
	}

	logger.Log.Info(fmt.Sprintf("Verified checksum %s (%d bytes)", checksum, written))
	if err := f.Chmod(0o755); err != nil {
		_ = os.Remove(binaryPath)
		return "", fmt.Errorf("failed to make binary executable: %w", err)
	}

	fmt.Printf("Installed %s (%d bytes) to %s\r\n", manifest.Name, written, binaryPath)
	logger.Log.Info(fmt.Sprintf("Installed module %s to %s", manifest.Name, binaryPath))
	return binaryPath, nil
}
