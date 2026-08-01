package module

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"time"

	"goscouter/internal/utils"

	"github.com/GoScouter/sdk"
	"github.com/GoScouter/sdk/style"
)

const infoTimeout = 5 * time.Second
const scoutTimeout = 4 * time.Minute
const closeTimeout = 5 * time.Second
const internalAuthor = "internal"

type External struct {
	Info     sdk.ModuleInfo
	Path     string
	Checksum string
}

func (e *External) Scout(ctx context.Context, target string, args []string) (json.RawMessage, string, error) {
	key := Key(Namespace(e.Info))

	checksum, err := utils.CalculateChecksum(e.Path)
	if err != nil {
		return nil, "", err
	}
	if checksum != e.Checksum {
		return nil, "", fmt.Errorf("%s changed on disk since it was loaded", key)
	}

	bin, err := sdk.Open(e.Path)
	if err != nil {
		return nil, "", err
	}
	defer shutdown(bin)

	ctx, cancel := context.WithTimeout(ctx, scoutTimeout)
	defer cancel()

	return bin.Scout(ctx, target, args)
}

type Manager struct {
	Internal  map[string]sdk.Module
	Externals map[string]*External

	Graph *Graph
}

func NewManager() *Manager {
	manager := &Manager{
		Internal:  make(map[string]sdk.Module),
		Externals: make(map[string]*External),
	}

	manager.AddInternal(&SubdomainsModule{})
	manager.AddInternal(&DnsModule{})
	manager.AddInternal(&HttpModule{})
	manager.AddInternal(&ScanModule{
		Manager: manager,
	})

	return manager
}

func (m *Manager) AddInternal(module sdk.Module) {
	m.Internal[Key(Namespace(module.Info()))] = module
}

func (m *Manager) Build() (*Graph, error) {
	infos := make([]sdk.ModuleInfo, 0, len(m.Internal)+len(m.Externals))

	for _, module := range m.Internal {
		infos = append(infos, module.Info())
	}
	for _, external := range m.Externals {
		infos = append(infos, external.Info)
	}

	graph, err := BuildGraph(infos)
	m.Graph = graph

	return graph, err
}

func (m *Manager) Lookup(key string) (sdk.ModuleInfo, bool) {
	if module, ok := m.Internal[key]; ok {
		return module.Info(), true
	}
	if external, ok := m.Externals[key]; ok {
		return external.Info, true
	}
	return sdk.ModuleInfo{}, false
}

func (m *Manager) RemoveInternal(name string) {
	delete(m.Internal, name)
}

func (m *Manager) GetInternals() []sdk.Module {
	return slices.Collect(maps.Values(m.Internal))
}

func (m *Manager) GetInternal(name string) sdk.Module {
	return m.Internal[name]
}

func (m *Manager) GetExternals() []*External {
	return slices.Collect(maps.Values(m.Externals))
}

func (m *Manager) GetExternal(name string) *External {
	return m.Externals[name]
}

func ExternalDir() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "gs", "modules"), nil
}

func (m *Manager) LoadExternals(ctx context.Context) error {
	dir, err := ExternalDir()
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)

	for _, entry := range entries {
		if !executable(entry) {
			continue
		}

		wg.Add(1)
		go func(entry os.DirEntry) {
			defer wg.Done()

			ex := open(ctx, filepath.Join(dir, entry.Name()))
			if ex == nil {
				return
			}

			key := Key(Namespace(ex.Info))

			mu.Lock()
			defer mu.Unlock()

			if _, taken := m.Internal[key]; taken {
				warn(fmt.Sprintf("skipping external module %q: %s is a built-in module", ex.Path, key))
				return
			}
			if other, taken := m.Externals[key]; taken {
				warn(fmt.Sprintf("skipping external module %q: %s is already provided by %s", ex.Path, key, other.Path))
				return
			}

			m.Externals[key] = ex
		}(entry)
	}

	wg.Wait()

	return nil
}

func open(ctx context.Context, path string) *External {
	bin, err := sdk.Open(path)
	if err != nil {
		warn(fmt.Sprintf("skipping external module %q: %v", path, err))
		return nil
	}
	defer shutdown(bin)

	ctx, cancel := context.WithTimeout(ctx, infoTimeout)
	defer cancel()

	info, err := bin.Info(ctx)
	if err != nil {
		warn(fmt.Sprintf("skipping external module %q: reading info: %v", path, err))
		return nil
	}

	if !info.IsValid() {
		warn(fmt.Sprintf("skipping external module %q: invalid info", path))
		return nil
	}

	checksum, err := utils.CalculateChecksum(path)
	if err != nil {
		warn(fmt.Sprintf("skipping external module %q: cannot calculate checksum", path))
		return nil
	}

	return &External{Path: path, Info: info, Checksum: checksum}
}

func shutdown(bin *sdk.Binary) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = bin.Close()
	}()

	select {
	case <-done:
	case <-time.After(closeTimeout):
		warn(fmt.Sprintf("module %q did not exit within %s, killing it", bin.Path(), closeTimeout))
		_ = bin.Kill()
		<-done
	}
}

func executable(entry os.DirEntry) bool {
	if entry.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return filepath.Ext(entry.Name()) == ".exe"
	}

	info, err := entry.Info()
	if err != nil {
		return false
	}
	return info.Mode().Perm()&0o111 != 0
}

// warn reports a problem the user can act on — a module that could not be
// loaded, or one that would not shut down — without aborting the session.
func warn(msg string) {
	fmt.Fprintf(os.Stderr, "%s\r\n", style.Alert(msg))
}
