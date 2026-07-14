package module

import (
	"fmt"
	"maps"
	"slices"

	"github.com/GoScouter/sdk"
)

type Manager struct {
    Modules map[string]sdk.Module
}

func NewManager() *Manager {
	m := &Manager{Modules: make(map[string]sdk.Module)}
	m.Add(&RecordsModule{})
	m.Add(&SubdomainsModule{})
    m.Add(&HttpModule{})
    m.Add(&ScanModule{})
    return m
}

func (m *Manager) Add(mod sdk.Module) {
	m.Modules[mod.Name()] = mod
}

func (m *Manager) Get(name string) (sdk.Module, error) {
	mod, ok := m.Modules[name]
	if !ok {
		return nil, fmt.Errorf("%s - module does not exist", name)
	}

	return mod, nil
}

func (m *Manager) GetAll() ([]sdk.Module) {
    return slices.Collect(maps.Values(m.Modules))
}
