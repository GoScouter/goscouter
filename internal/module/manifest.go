package module

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/src-d/go-git.v4"
)

type Manifest struct {
	Name     string            `json:"name"`
    Releases map[string]Release `json:"releases"`
}

type Release struct {
    Checksum string `json:"sha256"`
	Binary   string `json:"binary"`
}

var modulePattern = regexp.MustCompile(`^(?:https?://)?(.+?)@([^@]+)$`)

type ModuleRef struct {
	Git  string
	Version string
}

func ParseModule(s string) *ModuleRef {
	m := modulePattern.FindStringSubmatch(s)
	if m == nil {
		return nil
	}

	return &ModuleRef{
		Git:  m[1],
		Version: m[2],
	}
}

func normalizeGitURL(raw string) []string {
	if strings.Contains(raw, "://") || strings.Contains(raw, "@") {
		return []string{raw}
	}

	return []string{
		"https://" + raw,
		"http://" + raw,
	}
}

func ResolveManifest(ref *ModuleRef) (*Manifest, error) {
    if ref == nil {
        return nil, fmt.Errorf("module ref cannot be null")
    }

	tempDir, err := os.MkdirTemp("", "repo-*")
	if err != nil {
	    return nil, err
	}
	defer os.RemoveAll(tempDir)

    for _, url := range normalizeGitURL(ref.Git) {
        _, err = git.PlainClone(tempDir, false, &git.CloneOptions{
            URL: url,
        })

        if err == nil {
            break
        }
    }

    if err != nil {
        return nil ,err
    }

    manifestPath := filepath.Join(tempDir, "manifest.json")
	data, err := os.Open(manifestPath)
	if err != nil {
        return nil, err
	}
    defer data.Close()

    var manifest Manifest
	if err := json.NewDecoder(data).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %v", err)
	}

    return &manifest, nil
}
