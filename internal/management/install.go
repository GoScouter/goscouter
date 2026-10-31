package management

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"goscouter/internal/module"
	"goscouter/internal/utils"
	"goscouter/pkg/modules"

	"github.com/GoScouter/sdk"
	"github.com/GoScouter/sdk/style"
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
	artifacts, ok := vers.Artifacts[platform]
	if !ok {
		return fmt.Errorf("cannot find matching artifact for your architecture")
	}

	step("Resolved %s %s for %s", style.Bold(manifest.Name), style.Bold(version), platform)

	modulesDir, err := module.ExternalDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(modulesDir, 0o755); err != nil {
		return err
	}

	manager := module.NewManager()
	if err := manager.LoadExternals(ctx); err != nil {
		return err
	}

	in := &installer{
		dir:      modulesDir,
		manager:  manager,
		download: download,
		info:     moduleInfo,
	}
	return in.install(ctx, manifest.Name, version, artifacts)
}

type installer struct {
	dir      string
	manager  *module.Manager
	download func(ctx context.Context, url string) (io.ReadCloser, error)
	info     func(ctx context.Context, path string) (sdk.ModuleInfo, error)
}

type staged struct {
	path     string
	target   string
	key      string
	outdated string
}

func (in *installer) install(ctx context.Context, name, version string, artifacts modules.Artifacts) error {
	names := make(map[string]bool, len(artifacts))
	for _, a := range artifacts {
		filename := path.Base(a.Url)
		switch {
		case a.Url == "":
			return fmt.Errorf("manifest lists an artifact without a url")
		case a.Checksum == "":
			return fmt.Errorf("manifest lists no checksum for %s", filename)
		case names[filename]:
			return fmt.Errorf("manifest lists %s twice", filename)
		}
		names[filename] = true
	}

	var ready []*staged
	discard := func() {
		for _, s := range ready {
			os.Remove(s.path)
		}
	}

	keys := make(map[string]string, len(artifacts))
	for _, a := range artifacts {
		filename := path.Base(a.Url)

		s, err := in.stage(ctx, a, version)
		if err != nil {
			discard()
			if len(artifacts) > 1 {
				return fmt.Errorf("%s: %w\nnothing was installed", filename, err)
			}
			return err
		}
		if s == nil {
			continue
		}

		if other, taken := keys[s.key]; taken {
			os.Remove(s.path)
			discard()
			return fmt.Errorf("%s and %s are both %s, nothing was installed", other, filename, s.key)
		}
		keys[s.key] = filename
		ready = append(ready, s)
	}

	if len(ready) == 0 {
		return fmt.Errorf("%s %s is already installed", name, version)
	}

	for i, s := range ready {
		if err := os.Rename(s.path, s.target); err != nil {
			for _, rest := range ready[i:] {
				os.Remove(rest.path)
			}
			return err
		}
		if s.outdated != "" {
			os.Remove(s.outdated)
		}
		step("Installed %s %s to %s", style.Bold(s.key), style.Bold(version), style.Dim(s.target))
	}

	if len(artifacts) > 1 {
		step("Installed %d modules from %s %s", len(ready), style.Bold(name), style.Bold(version))
	}
	return nil
}

func (in *installer) stage(ctx context.Context, a modules.Artifact, version string) (*staged, error) {
	filename := path.Base(a.Url)
	target := filepath.Join(in.dir, filename)

	if installed, err := utils.CalculateChecksum(target); err == nil {
		if strings.EqualFold(installed, a.Checksum) {
			notice("%s is already installed at %s", filename, style.Dim(target))
			return nil, nil
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	step("Downloading %s", style.Dim(a.Url))
	body, err := in.download(ctx, a.Url)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	tmp := filepath.Join(in.dir, fmt.Sprintf(".%s.download.%d", filename, os.Getpid()))
	out, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, err
	}

	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(out, hash), body); err != nil {
		out.Close()
		os.Remove(tmp)
		return nil, err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return nil, err
	}

	got := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(got, a.Checksum) {
		os.Remove(tmp)
		return nil, fmt.Errorf(
			"checksum mismatch for %s\n  expected: %s\n  actual:   %s\nThe download may be corrupt or tampered with — not installing",
			filename, a.Checksum, got,
		)
	}

	step("Checksum verified %s", style.Dim(got))

	if err := os.Chmod(tmp, 0o755); err != nil {
		os.Remove(tmp)
		return nil, err
	}

	info, err := in.info(ctx, tmp)
	if err != nil {
		os.Remove(tmp)
		return nil, err
	}

	key := module.Key(module.Namespace(info))
	step("Module identified as %s", style.Bold(key))

	if in.manager.GetInternal(key) != nil {
		os.Remove(tmp)
		return nil, fmt.Errorf("%s is a built-in module", key)
	}

	s := &staged{path: tmp, target: target, key: key}
	if other := in.manager.GetExternal(key); other != nil && other.Path != target {
		if strings.EqualFold(other.Checksum, a.Checksum) {
			os.Remove(tmp)
			notice("%s %s is already installed at %s", key, version, style.Dim(other.Path))
			return nil, nil
		}
		s.outdated = other.Path
		notice("Replacing older install at %s", style.Dim(other.Path))
	}
	return s, nil
}

func download(ctx context.Context, url string) (io.ReadCloser, error) {
	resp, err := utils.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("no binary was found (HTTP %d)", resp.StatusCode)
	}
	return resp.Body, nil
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
