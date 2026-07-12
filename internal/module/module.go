package module

import (
	"fmt"

	"goscouter/internal/dns"
	"goscouter/internal/web"
)

type Module interface {
	Name() string
	Description() string
	Scout(target string) (*Records, error)
}

type Records struct {
	Target string
	Host   string
	DNS    *dns.Records
	HTTP   *web.HTTPRecords
}

type Manager struct {
	Modules map[string]Module
}

func NewManager() *Manager {
	m := &Manager{Modules: make(map[string]Module)}
	m.Add(&RecordsModule{})
	return m
}

func (m *Manager) Add(mod Module) {
	m.Modules[mod.Name()] = mod
}

func (m *Manager) Get(name string) (Module, error) {
	mod, ok := m.Modules[name]
	if !ok {
		return nil, fmt.Errorf("%s - module does not exist", name)
	}

	return mod, nil
}
