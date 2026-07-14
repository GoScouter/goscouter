package module

import (
    "encoding/json"
    "net/http"
    "fmt"
    "regexp"
)

type Manifest struct {
    Name string `json:"name"`
    Version string `json:"version"`
    Platforms map[string]Platform `json:"platforms"`
}

type Platform struct {
    Checksum string `json:"sha256"`
    Binary string `json:"binary"`
}

var modulePattern = regexp.MustCompile(`^([^/@]+)/([^/@]+)@([^/@]+)$`)

type ModuleRef struct {
    Author string
    Module string
    Version string
}

func (link *ModuleRef) ToString() string {
    return fmt.Sprintf("%s/%s@%s", link.Author, link.Module, link.Version)
}

func ParseModule(s string) (*ModuleRef) {
    m := modulePattern.FindStringSubmatch(s)
	if m == nil {
		return nil
    }

    return &ModuleRef{
        Author: m[1],
        Module: m[2],
        Version: m[3],
    }
}

func ResolveManifest(url string) (Manifest, error) {
    var manifest Manifest
    resp, err := http.Get(url)
    if err != nil {
        return manifest, err
    }

    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return manifest, fmt.Errorf("no manifest was found (HTTP %d)", resp.StatusCode)
    }

    if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
        return manifest, fmt.Errorf("failed to decode JSON: %v", err)
    }

    return manifest, nil
}
