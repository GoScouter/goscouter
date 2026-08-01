package modules

type Manifest struct {
	Name     string             `json:"name"`
	Latest   string             `json:"latest"`
	Versions map[string]Version `json:"versions"`
}

type Version struct {
	Artifacts map[string]Artifact `json:"artifacts"`
}

type Artifact struct {
	Url      string `json:"url"`
	Checksum string `json:"checksum"`
}
