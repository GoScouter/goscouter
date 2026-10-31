package management

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goscouter/internal/module"
	"goscouter/pkg/modules"

	"github.com/GoScouter/sdk"
)

// fakeRelease serves "binaries" whose content is the module key they report.
type fakeRelease struct {
	files     map[string]string
	downloads int
}

func (f *fakeRelease) download(_ context.Context, url string) (io.ReadCloser, error) {
	f.downloads++
	body, ok := f.files[url]
	if !ok {
		return nil, errors.New("no binary was found (HTTP 404)")
	}
	return io.NopCloser(bytes.NewBufferString(body)), nil
}

func fakeInfo(_ context.Context, path string) (sdk.ModuleInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return sdk.ModuleInfo{}, err
	}
	author, name, _ := strings.Cut(string(data), ":")
	return sdk.ModuleInfo{Author: author, Name: name, Description: "test module"}, nil
}

func checksum(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func artifact(url, content string) modules.Artifact {
	return modules.Artifact{Url: url, Checksum: checksum(content)}
}

func newInstaller(t *testing.T, release *fakeRelease) *installer {
	t.Helper()
	return &installer{
		dir:      t.TempDir(),
		manager:  module.NewManager(),
		download: release.download,
		info:     fakeInfo,
	}
}

func recon() (*fakeRelease, modules.Artifacts) {
	release := &fakeRelease{files: map[string]string{
		"https://x/gs-recon-ports":  "recon:ports",
		"https://x/gs-recon-vhosts": "recon:vhosts",
		"https://x/gs-recon-dirs":   "recon:dirs",
	}}
	artifacts := modules.Artifacts{
		artifact("https://x/gs-recon-ports", "recon:ports"),
		artifact("https://x/gs-recon-vhosts", "recon:vhosts"),
		artifact("https://x/gs-recon-dirs", "recon:dirs"),
	}
	return release, artifacts
}

func dirEntries(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func expectInstalled(t *testing.T, dir string, want ...string) {
	t.Helper()
	if got := strings.Join(dirEntries(t, dir), ","); got != strings.Join(want, ",") {
		t.Fatalf("expected the modules dir to hold %v, got %q", want, got)
	}
}

func TestInstallSeveralModules(t *testing.T) {
	release, artifacts := recon()
	in := newInstaller(t, release)

	if err := in.install(context.Background(), "recon", "v1.0.0", artifacts); err != nil {
		t.Fatal(err)
	}
	expectInstalled(t, in.dir, "gs-recon-dirs", "gs-recon-ports", "gs-recon-vhosts")

	content, _ := os.ReadFile(filepath.Join(in.dir, "gs-recon-vhosts"))
	if string(content) != "recon:vhosts" {
		t.Fatalf("installed the wrong binary: %q", content)
	}
}

func TestInstallSingleArtifact(t *testing.T) {
	release := &fakeRelease{files: map[string]string{"https://x/gs-nginx": "goscouter:nginx"}}
	in := newInstaller(t, release)

	err := in.install(context.Background(), "nginx", "v3.0.0", modules.Artifacts{artifact("https://x/gs-nginx", "goscouter:nginx")})
	if err != nil {
		t.Fatal(err)
	}
	expectInstalled(t, in.dir, "gs-nginx")
}

func TestInstallNothingWhenOneFails(t *testing.T) {
	cases := map[string]func(*fakeRelease, modules.Artifacts){
		"bad checksum": func(_ *fakeRelease, a modules.Artifacts) {
			a[1].Checksum = checksum("something else")
		},
		"missing download": func(r *fakeRelease, _ modules.Artifacts) {
			delete(r.files, "https://x/gs-recon-dirs")
		},
		"built-in name": func(r *fakeRelease, a modules.Artifacts) {
			r.files["https://x/gs-recon-dirs"] = "internal:dns"
			a[2].Checksum = checksum("internal:dns")
		},
		"two binaries, one module": func(r *fakeRelease, a modules.Artifacts) {
			r.files["https://x/gs-recon-dirs"] = "recon:ports"
			a[2].Checksum = checksum("recon:ports")
		},
	}

	for name, breakIt := range cases {
		t.Run(name, func(t *testing.T) {
			release, artifacts := recon()
			breakIt(release, artifacts)
			in := newInstaller(t, release)

			err := in.install(context.Background(), "recon", "v1.0.0", artifacts)
			if err == nil {
				t.Fatal("expected the install to fail")
			}
			if !strings.Contains(err.Error(), "nothing was installed") {
				t.Fatalf("expected the error to say nothing was installed, got %v", err)
			}
			// No modules and no leftover downloads.
			expectInstalled(t, in.dir)
		})
	}
}

func TestInstallRejectsBadManifest(t *testing.T) {
	cases := map[string]modules.Artifacts{
		"no checksum": {{Url: "https://x/gs-recon-ports"}},
		"no url":      {{Checksum: "aa"}},
		"same file twice": {
			artifact("https://x/gs-recon-ports", "recon:ports"),
			artifact("https://y/gs-recon-ports", "recon:ports"),
		},
	}
	for name, artifacts := range cases {
		t.Run(name, func(t *testing.T) {
			release, _ := recon()
			in := newInstaller(t, release)
			if err := in.install(context.Background(), "recon", "v1.0.0", artifacts); err == nil {
				t.Fatal("expected an error")
			}
			if release.downloads != 0 {
				t.Fatalf("expected nothing downloaded, got %d downloads", release.downloads)
			}
		})
	}
}

func TestInstallAlreadyInstalled(t *testing.T) {
	release, artifacts := recon()
	in := newInstaller(t, release)
	if err := in.install(context.Background(), "recon", "v1.0.0", artifacts); err != nil {
		t.Fatal(err)
	}

	release.downloads = 0
	err := in.install(context.Background(), "recon", "v1.0.0", artifacts)
	if err == nil || !strings.Contains(err.Error(), "already installed") {
		t.Fatalf("expected an already installed error, got %v", err)
	}
	if release.downloads != 0 {
		t.Fatalf("expected nothing downloaded again, got %d downloads", release.downloads)
	}
}

func TestInstallOnlyWhatIsMissing(t *testing.T) {
	release, artifacts := recon()
	in := newInstaller(t, release)
	if err := in.install(context.Background(), "recon", "v1.0.0", artifacts[:1]); err != nil {
		t.Fatal(err)
	}

	release.downloads = 0
	if err := in.install(context.Background(), "recon", "v1.0.0", artifacts); err != nil {
		t.Fatal(err)
	}
	if release.downloads != 2 {
		t.Fatalf("expected only the two missing modules downloaded, got %d", release.downloads)
	}
	expectInstalled(t, in.dir, "gs-recon-dirs", "gs-recon-ports", "gs-recon-vhosts")
}

func TestInstallReplacesOlderVersion(t *testing.T) {
	release, artifacts := recon()
	in := newInstaller(t, release)

	old := filepath.Join(in.dir, "gs-recon-ports-v0")
	if err := os.WriteFile(old, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	in.manager.Externals["recon:ports"] = &module.External{
		Info:     sdk.ModuleInfo{Author: "recon", Name: "ports", Description: "old"},
		Path:     old,
		Checksum: checksum("old"),
	}

	if err := in.install(context.Background(), "recon", "v1.0.0", artifacts); err != nil {
		t.Fatal(err)
	}
	expectInstalled(t, in.dir, "gs-recon-dirs", "gs-recon-ports", "gs-recon-vhosts")
}
