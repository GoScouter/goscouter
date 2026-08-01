package management

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"goscouter/internal/module"
	"goscouter/internal/style"
	"goscouter/internal/utils"
	"goscouter/pkg/modules"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/GoScouter/sdk"
	"github.com/go-git/go-git/v5"
)

var modulePattern = regexp.MustCompile(`^(https?://[^@]+)(?:@([^@]+))?$`)

const infoTimeout = 5 * time.Second

func InstallModule(ctx context.Context, input string) error {
	matches := modulePattern.FindStringSubmatch(input)
	if matches == nil {
		return fmt.Errorf("invalid module")
	}

	url := matches[1]
	version := matches[2]

	dir, err := os.MkdirTemp("/tmp", "module-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	step("Fetching manifest from %s", style.Bold(url))
	_, err = git.PlainClone(dir, false, &git.CloneOptions{
		URL: url,
	})
	if err != nil {
		return err
	}

	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return err
	}

	var manifest modules.Manifest
	err = json.Unmarshal(data, &manifest)
	if err != nil {
		return err
	}

	if version == "latest" || version == "" {
		version = manifest.Latest
	}

	vers, ok := manifest.Versions[version]
	if !ok {
		return fmt.Errorf("cannot find matching version (%s)", version)
	}

	platform := runtime.GOOS + "-" + runtime.GOARCH
	artifact, ok := vers.Artifacts[platform]
	if !ok {
		return fmt.Errorf("cannot find matching artifact for your architecture")
	}

	if artifact.Checksum == "" {
		return fmt.Errorf("manifest lists no checksum for %s", platform)
	}

	step("Resolved %s %s for %s", style.Bold(manifest.Name), style.Bold(version), platform)

	modulesDir, err := module.ExternalDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(modulesDir, 0o755); err != nil {
		return err
	}

	filename := path.Base(artifact.Url)
	filePath := filepath.Join(modulesDir, filename)

	if installed, err := utils.CalculateChecksum(filePath); err == nil {
		if strings.EqualFold(installed, artifact.Checksum) {
			return fmt.Errorf("%s %s is already installed at %s", manifest.Name, version, filePath)
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	step("Downloading %s", style.Dim(artifact.Url))
	resp, err := http.Get(artifact.Url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("no binary was found (HTTP %d)", resp.StatusCode)
	}

	staged := filepath.Join(modulesDir, fmt.Sprintf(".%s.download.%d", filename, os.Getpid()))
	out, err := os.OpenFile(staged, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}

	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(out, hash), resp.Body); err != nil {
		out.Close()
		os.Remove(staged)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(staged)
		return err
	}

	got := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(got, artifact.Checksum) {
		os.Remove(staged)
		return fmt.Errorf(
			"checksum mismatch for %s\n  expected: %s\n  actual:   %s\nThe download may be corrupt or tampered with — not installing",
			filename, artifact.Checksum, got,
		)
	}

	step("Checksum verified %s", style.Dim(got))

	manager := module.NewManager()
	if err := manager.LoadExternals(ctx); err != nil {
		os.Remove(staged)
		return err
	}

	if err := os.Chmod(staged, 0o755); err != nil {
		os.Remove(staged)
		return err
	}

	info, err := moduleInfo(ctx, staged)
	if err != nil {
		os.Remove(staged)
		return err
	}

	key := module.Key(module.Namespace(info))
	step("Module identified as %s", style.Bold(key))

	if manager.GetInternal(key) != nil {
		os.Remove(staged)
		return fmt.Errorf("%s is a built-in module", key)
	}

	var outdated string
	if other := manager.GetExternal(key); other != nil && other.Path != filePath {
		if strings.EqualFold(other.Checksum, artifact.Checksum) {
			os.Remove(staged)
			return fmt.Errorf("%s %s is already installed at %s", key, version, other.Path)
		}
		outdated = other.Path
		notice("Replacing older install at %s", style.Dim(other.Path))
	}

	if err := os.Rename(staged, filePath); err != nil {
		os.Remove(staged)
		return err
	}

	if outdated != "" {
		os.Remove(outdated)
	}

	step("Installed %s %s to %s", style.Bold(key), style.Bold(version), style.Dim(filePath))

	return nil
}

func step(format string, a ...any) {
	fmt.Printf("%s\r\n", style.Foundf(format, a...))
}

func notice(format string, a ...any) {
	fmt.Printf("%s\r\n", style.Alertf(format, a...))
}

func moduleInfo(ctx context.Context, path string) (sdk.ModuleInfo, error) {
	bin, err := sdk.Open(path)
	if err != nil {
		return sdk.ModuleInfo{}, err
	}
	defer bin.Close()

	ctx, cancel := context.WithTimeout(ctx, infoTimeout)
	defer cancel()

	info, err := bin.Info(ctx)
	if err != nil {
		return sdk.ModuleInfo{}, fmt.Errorf("reading info: %w", err)
	}
	if !info.IsValid() {
		return sdk.ModuleInfo{}, fmt.Errorf("%s reports invalid info", filepath.Base(path))
	}

	return info, nil
}
