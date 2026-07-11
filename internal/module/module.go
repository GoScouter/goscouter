package module

import (
	"fmt"

	"goscouter/internal/dns"
	"goscouter/internal/web"
)

// Module is a unit of reconnaissance that gathers information about a target.
//
// Builtin modules (like RecordsModule) are implemented directly in Go. External
// modules will be standalone executables driven over the SDK; they will adapt to
// this same interface so the shell can treat both kinds uniformly.
type Module interface {
	Name() string
	Description() string
	Scout(target string) (*Records, error)
}

// Records is the structured output a module produces for a target. Keeping this
// as plain data (no presentation) lets the shell render builtin and external
// module output the same way.
type Records struct {
	Target string
	Host   string
	DNS    *dns.Records
	HTTP   *web.HTTPRecords
}

// Manager holds the set of registered modules.
type Manager struct {
	Modules map[string]Module
}

// NewManager returns a Manager preloaded with the builtin modules.
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
