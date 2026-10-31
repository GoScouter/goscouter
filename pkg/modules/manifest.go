package modules

import (
	"bytes"
	"encoding/json"
	"errors"
)

type Manifest struct {
	Name     string             `json:"name"`
	Latest   string             `json:"latest"`
	Versions map[string]Version `json:"versions"`
}

type Version struct {
	Artifacts map[string]Artifacts `json:"artifacts"`
}

type Artifact struct {
	Url      string `json:"url"`
	Checksum string `json:"checksum"`
}

// Artifacts are the binaries a platform installs. A manifest writes a single
// binary as one object, the original format, and several as a list.
type Artifacts []Artifact

func (a *Artifacts) UnmarshalJSON(data []byte) error {
	if trimmed := bytes.TrimSpace(data); len(trimmed) > 0 && trimmed[0] == '[' {
		var list []Artifact
		if err := json.Unmarshal(data, &list); err != nil {
			return err
		}
		if len(list) == 0 {
			return errors.New("empty artifact list")
		}
		*a = list
		return nil
	}

	var one Artifact
	if err := json.Unmarshal(data, &one); err != nil {
		return err
	}
	*a = Artifacts{one}
	return nil
}
