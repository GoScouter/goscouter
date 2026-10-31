package modules

import (
	"encoding/json"
	"testing"
)

func TestArtifactsSingleObject(t *testing.T) {
	var v Version
	err := json.Unmarshal([]byte(`{"artifacts": {"linux-amd64": {"url": "https://x/a", "checksum": "aa"}}}`), &v)
	if err != nil {
		t.Fatal(err)
	}
	got := v.Artifacts["linux-amd64"]
	if len(got) != 1 || got[0].Url != "https://x/a" || got[0].Checksum != "aa" {
		t.Fatalf("got %+v", got)
	}
}

func TestArtifactsList(t *testing.T) {
	var v Version
	err := json.Unmarshal([]byte(`{"artifacts": {"linux-amd64": [
		{"url": "https://x/ports", "checksum": "aa"},
		{"url": "https://x/dirs", "checksum": "bb"}
	]}}`), &v)
	if err != nil {
		t.Fatal(err)
	}
	got := v.Artifacts["linux-amd64"]
	if len(got) != 2 || got[0].Url != "https://x/ports" || got[1].Checksum != "bb" {
		t.Fatalf("got %+v", got)
	}
}

func TestArtifactsInvalid(t *testing.T) {
	for name, input := range map[string]string{
		"empty list":   `[]`,
		"not artifact": `"https://x/a"`,
		"bad element":  `[1, 2]`,
	} {
		var a Artifacts
		if err := json.Unmarshal([]byte(input), &a); err == nil {
			t.Errorf("%s: expected an error, got %+v", name, a)
		}
	}
}
