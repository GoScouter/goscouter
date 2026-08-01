package module

import (
	"errors"
	"fmt"
	"slices"

	"github.com/GoScouter/sdk"
	"github.com/stevenle/topsort"
)

func Key(ns sdk.ModuleNamespace) string {
	return fmt.Sprintf("%s:%s", ns.Author, ns.Name)
}

func Namespace(info sdk.ModuleInfo) sdk.ModuleNamespace {
	return sdk.ModuleNamespace{Name: info.Name, Author: info.Author}
}

func IsInternalAuthor(author string) bool {
	return author == internalAuthor
}

const everything = "*"

func scanKey() string {
	return Key(Namespace(ScanModuleInfo))
}

type Graph struct {
	sorter *topsort.Graph
	known  map[string]bool
}

func BuildGraph(infos []sdk.ModuleInfo) (*Graph, error) {
	g := &Graph{sorter: topsort.NewGraph(), known: make(map[string]bool, len(infos))}

	for _, info := range infos {
		if info.Author == internalAuthor && info.Name == SdModuleInfo.Name {
			continue
		}
		g.known[Key(Namespace(info))] = true
	}

	var problems []error

	for _, info := range infos {
		if info.Author == internalAuthor && info.Name == SdModuleInfo.Name {
			continue
		}

		key := Key(Namespace(info))
		g.sorter.AddEdge(everything, key)

		for _, dep := range info.Dependencies {
			if !g.known[Key(dep)] {
				problems = append(problems, missing(key, dep))
				continue
			}
			g.sorter.AddEdge(key, Key(dep))
		}
	}

	if _, err := g.Order(); err != nil {
		problems = append(problems, err)
	}

	return g, errors.Join(problems...)
}

func (g *Graph) Plan(key string) ([]string, error) {
	if !g.known[key] {
		return nil, fmt.Errorf("unknown module %q", key)
	}

	plan, err := g.sorter.TopSort(key)
	if err != nil {
		return nil, err
	}

	// scan runs every other module itself, so it stands in for whatever the
	// requested module depends on. The module itself still runs, last.
	if key != scanKey() && slices.Contains(plan, scanKey()) {
		return []string{scanKey(), key}, nil
	}

	return plan, nil
}

func (g *Graph) Order() ([]string, error) {
	order, err := g.sorter.TopSort(everything)
	if err != nil {
		return nil, err
	}

	order = order[:len(order)-1]

	// scan is what walks this order, so neither scan nor anything that depends on it belongs in it.
	out := make([]string, 0, len(order))
	for _, key := range order {
		plan, err := g.Plan(key)
		if err != nil {
			return nil, err
		}
		if slices.Contains(plan, scanKey()) {
			continue
		}
		out = append(out, key)
	}

	return out, nil
}

func missing(key string, dep sdk.ModuleNamespace) error {
	return fmt.Errorf("%s depends on %s, which is not installed", key, Key(dep))
}
