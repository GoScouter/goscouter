package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/GoScouter/sdk/style"
)

type Result struct {
	Command string
	Target  string
	Time    time.Time
	Data    []json.RawMessage
	Views   []string
}

type SaveCommand struct {
	Manager *CommandManager
}

func (cmd *SaveCommand) Name() string {
	return "save"
}

func (cmd *SaveCommand) Description() string {
	return "Saves the last module result to a .json, .md or .txt file"
}

func (cmd *SaveCommand) Usage() string {
	return "save <file.json|file.md|file.txt>"
}

var saveExtensions = []string{".json", ".md", ".txt"}

func (cmd *SaveCommand) CompleteArgs(args []string, word string) []string {
	if len(args) > 0 {
		return nil
	}

	base := strings.TrimSuffix(word, filepath.Ext(word))
	if base == "" {
		base = "results"
	}

	var cands []string
	for _, ext := range saveExtensions {
		if name := base + ext; strings.HasPrefix(name, word) {
			cands = append(cands, name)
		}
	}
	return cands
}

func (cmd *SaveCommand) Exec(_ context.Context, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: %s", cmd.Usage())
	}

	last := cmd.Manager.Last
	if last == nil {
		return errors.New("nothing to save yet, run a module first")
	}

	path := args[0]
	var content []byte
	var err error
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		content, err = resultJSON(last)
	case ".md":
		content = resultMarkdown(last)
	default:
		content = []byte(resultText(last) + "\n")
	}
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, content, 0o644); err != nil {
		return err
	}

	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	fmt.Printf("%s\r\n", style.Successf("Saved %s results to %s", style.Bold(last.Command), style.Bold(path)))
	return nil
}

func resultJSON(r *Result) ([]byte, error) {
	var raw json.RawMessage
	if len(r.Data) == 1 {
		raw = r.Data[0]
	} else {
		all, err := json.Marshal(r.Data)
		if err != nil {
			return nil, err
		}
		raw = all
	}

	var out bytes.Buffer
	if err := json.Indent(&out, raw, "", "  "); err != nil {
		return nil, err
	}
	out.WriteByte('\n')
	return out.Bytes(), nil
}

func resultMarkdown(r *Result) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "# goscouter: %s\n\n", r.Command)
	fmt.Fprintf(&b, "- **Target:** %s\n", r.Target)
	fmt.Fprintf(&b, "- **Date:** %s\n\n", r.Time.Format(time.RFC1123))
	fmt.Fprintf(&b, "```text\n%s\n```\n", resultText(r))
	return []byte(b.String())
}

var ansiRE = regexp.MustCompile("\x1b\\[[0-9;?]*[A-Za-z]")

func resultText(r *Result) string {
	text := ansiRE.ReplaceAllString(strings.Join(r.Views, ""), "")
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.Trim(text, "\n")
}
