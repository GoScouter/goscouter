package module

import "fmt"

type Module interface {
	Name() string
	Description() string
	Scout(target string) (Result, error)
}

type Result interface {
	Render() string
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
