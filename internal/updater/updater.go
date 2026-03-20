package updater

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

const (
	repo          = "FlipsideCrypto/near-intents-cli"
	checkInterval = 24 * time.Hour
)

type githubRelease struct {
	TagName string `json:"tag_name"`
}

// LatestVersion fetches the latest release tag from GitHub.
func LatestVersion() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var r githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", err
	}
	if r.TagName == "" {
		return "", fmt.Errorf("no release found")
	}
	return r.TagName, nil
}

// IsNewer returns true if latest is a different version than current.
// Returns false when current is "dev" (local builds).
func IsNewer(current, latest string) bool {
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")
	return current != "dev" && latest != "" && latest != current
}

// throttleFile returns the path used to track when we last checked for updates.
func throttleFile(binary string) string {
	dir := filepath.Join(os.Getenv("HOME"), ".local", "share", binary)
	_ = os.MkdirAll(dir, 0755)
	return filepath.Join(dir, ".last_update_check")
}

// ShouldCheck returns true if 24h have elapsed since the last update check.
func ShouldCheck(binary string) bool {
	info, err := os.Stat(throttleFile(binary))
	if err != nil {
		return true
	}
	return time.Since(info.ModTime()) > checkInterval
}

// MarkChecked updates the throttle file timestamp.
func MarkChecked(binary string) {
	f := throttleFile(binary)
	_ = os.WriteFile(f, []byte(time.Now().Format(time.RFC3339)), 0644)
}

// Update downloads the given version of binary and replaces the running executable.
// On success it re-execs the process so the new binary takes over transparently.
func Update(binary, version string) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine executable path: %w", err)
	}
	// Resolve symlinks so we replace the real binary.
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("cannot resolve executable path: %w", err)
	}

	versionNum := strings.TrimPrefix(version, "v")
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	archive := fmt.Sprintf("%s_%s_%s_%s.tar.gz", binary, versionNum, goos, goarch)
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, version, archive)

	// Download archive to a temp file.
	tmp, err := os.CreateTemp("", binary+"-update-*.tar.gz")
	if err != nil {
		return fmt.Errorf("cannot create temp file: %w", err)
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d for %s", resp.StatusCode, url)
	}
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	tmp.Close()

	// Extract the binary from the archive.
	extracted, err := extractBinary(tmp.Name(), binary, filepath.Dir(execPath))
	if err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}
	defer os.Remove(extracted)

	// Atomically replace the running executable.
	if err := os.Rename(extracted, execPath); err != nil {
		return fmt.Errorf("failed to replace binary: %w", err)
	}
	if err := os.Chmod(execPath, 0755); err != nil {
		return fmt.Errorf("failed to chmod binary: %w", err)
	}

	return nil
}

// extractBinary pulls the named binary out of a tar.gz archive into destDir.
func extractBinary(archivePath, binaryName, destDir string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if filepath.Base(hdr.Name) != binaryName {
			continue
		}
		dest := filepath.Join(destDir, binaryName+".new")
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			os.Remove(dest)
			return "", err
		}
		out.Close()
		return dest, nil
	}
	return "", fmt.Errorf("binary %q not found in archive", binaryName)
}

// Reexec replaces the current process with a fresh invocation of the same binary + args.
func Reexec() error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}
	return syscall.Exec(execPath, os.Args, os.Environ())
}

// AutoCheck checks for a newer version at most once per 24h. If one is found it
// updates the binary in place, prints a notice to stderr, and re-execs the process.
// Errors are silently ignored so a network hiccup never breaks the CLI.
func AutoCheck(binary, currentVersion string) {
	if !ShouldCheck(binary) {
		return
	}
	MarkChecked(binary)

	latest, err := LatestVersion()
	if err != nil || !IsNewer(currentVersion, latest) {
		return
	}

	fmt.Fprintf(os.Stderr, "Updating %s %s → %s...\n", binary, currentVersion, latest)
	if err := Update(binary, latest); err != nil {
		fmt.Fprintf(os.Stderr, "Auto-update failed: %v\n", err)
		return
	}
	fmt.Fprintf(os.Stderr, "Updated. Restarting...\n\n")
	// Re-exec with the new binary — transparent to the user.
	if err := Reexec(); err != nil {
		fmt.Fprintf(os.Stderr, "Restart failed: %v — please re-run your command.\n", err)
	}
}

// RunUpdate is the handler for the explicit `update` subcommand.
func RunUpdate(binary, currentVersion string) error {
	fmt.Fprintf(os.Stderr, "Checking for updates...\n")
	latest, err := LatestVersion()
	if err != nil {
		return fmt.Errorf("could not fetch latest version: %w", err)
	}

	if !IsNewer(currentVersion, latest) {
		fmt.Printf("%s is already up to date (%s)\n", binary, currentVersion)
		return nil
	}

	fmt.Fprintf(os.Stderr, "Updating %s %s → %s...\n", binary, currentVersion, latest)
	if err := Update(binary, latest); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}
	MarkChecked(binary)
	fmt.Printf("Updated %s to %s\n", binary, latest)
	return nil
}
