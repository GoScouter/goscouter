package terminal

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

const maxHistory = 1000

func HistoryPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gs", "history"), nil
}

func loadHistory(path string) ([]string, error) {
	if path == "" {
		return nil, nil
	}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var history []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" {
			history = append(history, line)
		}
	}
	return history, scanner.Err()
}

func appendHistory(path, cmd string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(cmd + "\n")
	return err
}

func saveHistory(path string, history []string) error {
	return os.WriteFile(path, []byte(strings.Join(history, "\n")+"\n"), 0o600)
}
