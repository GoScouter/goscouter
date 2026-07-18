package versions

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"goscouter/internal/logger"
	"goscouter/internal/style"

	"github.com/google/go-github/github"
)

const UPDATE_SUGGESTION = `
⚠️  Update Available: %s → %s

A newer version is available and may contain important
security fixes and improvements.
`

const (
	OWNER  = "GoScouter"
	REPO   = "goscouter"
	BINARY = "gs"

	CHECKSUMS = "checksums.txt"
)

const (
	releaseTimeout  = 15 * time.Second
	downloadTimeout = 5 * time.Minute
)

func printBox(lines []string) {
	maxLen := 0
	for _, line := range lines {
		if len(line) > maxLen {
			maxLen = len(line)
		}
	}

	border := "╔" + strings.Repeat("═", maxLen+2) + "╗"
	bottom := "╚" + strings.Repeat("═", maxLen+2) + "╝"

	fmt.Println(border)
	for _, line := range lines {
		fmt.Printf("║ %-*s ║\n", maxLen, line)
	}
	fmt.Println(bottom)
}

func assetName() string {
	name := fmt.Sprintf("%s-%s-%s", BINARY, runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

func sameVersion(a, b string) bool {
	return strings.TrimPrefix(a, "v") == strings.TrimPrefix(b, "v")
}

func confirm(reader *bufio.Reader, prompt string) bool {
	fmt.Printf("%s %s ", prompt, style.Dim("[y/N]:"))

	answer, err := reader.ReadString('\n')
	if err != nil {
		// EOF or a read error: treat silence as "no".
		fmt.Println()
		return false
	}

	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes"
}

func findAsset(release *github.RepositoryRelease, name string) (github.ReleaseAsset, error) {
	for _, asset := range release.Assets {
		if asset.GetName() == name {
			return asset, nil
		}
	}
	return github.ReleaseAsset{}, fmt.Errorf("release %s publishes no %s asset", release.GetTagName(), name)
}

func openAsset(ctx context.Context, client *github.Client, asset github.ReleaseAsset) (io.ReadCloser, error) {
	body, redirect, err := client.Repositories.DownloadReleaseAsset(ctx, OWNER, REPO, asset.GetID())
	if err != nil {
		return nil, fmt.Errorf("cannot download %s: %w", asset.GetName(), err)
	}
	if body != nil {
		return body, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, redirect, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("cannot download %s: %s", asset.GetName(), resp.Status)
	}

	return resp.Body, nil
}

func expectedChecksum(ctx context.Context, client *github.Client, release *github.RepositoryRelease, name string) (string, error) {
	asset, err := findAsset(release, CHECKSUMS)
	if err != nil {
		return "", err
	}

	body, err := openAsset(ctx, client, asset)
	if err != nil {
		return "", err
	}
	defer body.Close()

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && strings.TrimPrefix(fields[1], "*") == name {
			return fields[0], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("%s lists no checksum for %s", CHECKSUMS, name)
}

func download(ctx context.Context, client *github.Client, release *github.RepositoryRelease, exe string) (string, error) {
	name := assetName()

	asset, err := findAsset(release, name)
	if err != nil {
		return "", err
	}

	want, err := expectedChecksum(ctx, client, release, name)
	if err != nil {
		return "", err
	}

	staged := filepath.Join(filepath.Dir(exe), fmt.Sprintf(".%s.update.%d", BINARY, os.Getpid()))
	file, err := os.OpenFile(staged, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o755)
	if err != nil {
		return "", fmt.Errorf("cannot stage the download in %s: %w", filepath.Dir(exe), err)
	}

	body, err := openAsset(ctx, client, asset)
	if err != nil {
		file.Close()
		os.Remove(staged)
		return "", err
	}
	defer body.Close()

	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(file, hash), body); err != nil {
		file.Close()
		os.Remove(staged)
		return "", err
	}
	if err := file.Close(); err != nil {
		os.Remove(staged)
		return "", err
	}

	got := hex.EncodeToString(hash.Sum(nil))
	if got != want {
		os.Remove(staged)
		return "", fmt.Errorf(
			"checksum mismatch for %s\n  expected: %s\n  actual:   %s\nThe download may be corrupt or tampered with — not installing",
			name, want, got,
		)
	}

	return staged, nil
}

func install(staged, exe string) error {
	backup := exe + ".old"
	os.Remove(backup)

	if err := os.Rename(exe, backup); err != nil {
		return fmt.Errorf("cannot move %s aside: %w", exe, err)
	}

	if err := os.Rename(staged, exe); err != nil {
		os.Rename(backup, exe)
		return fmt.Errorf("cannot install to %s: %w", exe, err)
	}

	os.Remove(backup)
	return nil
}

func SuggestUpdate(current string) error {
	if current == "" || current == "dev" {
		logger.Log.Info("Skipping update check for an unversioned build")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), releaseTimeout)
	defer cancel()

	client := github.NewClient(nil)
	release, _, err := client.Repositories.GetLatestRelease(ctx, OWNER, REPO)
	if err != nil {
		return err
	}

	latest := release.GetTagName()
	if sameVersion(current, latest) {
		return nil
	}

	text := fmt.Sprintf(UPDATE_SUGGESTION, current, latest)
	printBox(strings.Split(text, "\n"))

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if exe, err = filepath.EvalSymlinks(exe); err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)
	if !confirm(reader, fmt.Sprintf("Download %s for %s/%s?", latest, runtime.GOOS, runtime.GOARCH)) {
		return nil
	}

	downloadCtx, cancelDownload := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancelDownload()

	fmt.Printf("%s\n", style.Dim("Downloading "+assetName()+"..."))
	staged, err := download(downloadCtx, client, release, exe)
	if err != nil {
		return err
	}
	defer os.Remove(staged)

	fmt.Printf("%s\n", style.Dim("Checksum verified."))
	if !confirm(reader, fmt.Sprintf("Install %s over %s?", latest, exe)) {
		fmt.Printf("%s\n\n", style.Info("Update downloaded but not installed."))
		return nil
	}

	if err := install(staged, exe); err != nil {
		return err
	}

	fmt.Printf("%s\n", style.Success(fmt.Sprintf("Updated to %s. Run %s again to use it.", latest, BINARY)))
	os.Exit(0)

	return nil
}
