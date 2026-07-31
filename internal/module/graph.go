package module

import (
	"errors"
	"fmt"

	"github.com/GoScouter/sdk"
	"github.com/stevenle/topsort"
)

func Key(ns sdk.ModuleNamespace) string {
	return fmt.Sprintf("%s:%s", ns.Author, ns.Name)
}

func Namespace(info sdk.ModuleInfo) sdk.ModuleNamespace {
	return sdk.ModuleNamespace{Name: info.Name, Author: info.Author}
}

const everything = "*"

type Graph struct {
	sorter *topsort.Graph
	known  map[string]bool
}

func BuildGraph(infos []sdk.ModuleInfo) (*Graph, error) {
	g := &Graph{sorter: topsort.NewGraph(), known: make(map[string]bool, len(infos))}

	for _, info := range infos {
		if info.Author == internalAuthor && info.Name == "subdomains" {
			continue
		}
		g.known[Key(Namespace(info))] = true
	}

	var problems []error

	for _, info := range infos {
		if info.Author == internalAuthor && info.Name == "subdomains" {
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
	return g.sorter.TopSort(key)
}

func (g *Graph) Order() ([]string, error) {
	order, err := g.sorter.TopSort(everything)
	if err != nil {
		return nil, err
	}
	return order[:len(order)-1], nil
}

func missing(key string, dep sdk.ModuleNamespace) error {
	return fmt.Errorf("%s depends on %s, which is not installed", key, Key(dep))
}
