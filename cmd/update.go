package cmd

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"bookleaf-cli/internal/format"

	"github.com/spf13/cobra"
)

const (
	ghOwner    = "atharva-again"
	ghRepo     = "bookleaf-cli"
	binaryName = "bookleaf"
)

var updateCmd = &cobra.Command{
	Use:   "update [version]",
	Short: "Update the bookleaf binary to a newer version",
	Long: `Update the bookleaf CLI to the latest or a specific version.

Keywords:
  latest          Update to the latest stable release (default)
  canary          Update to the latest available release

Examples:
  bookleaf update          update to latest
  bookleaf update latest   update to latest
  bookleaf update v0.1.1   update to a specific version
  bookleaf update canary   update to latest (canary not configured yet)`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		targetVersion := ""

		if len(args) == 0 || args[0] == "latest" || args[0] == "canary" {
			if args[0] == "canary" {
				fmt.Println("Note: canary builds are not configured. Falling back to latest release.")
			}
			ver, err := fetchLatestVersion()
			if err != nil {
				return fmt.Errorf("fetch latest version: %w", err)
			}
			targetVersion = ver
		} else {
			targetVersion = strings.TrimPrefix(args[0], "v")
		}

		currentVersion := strings.TrimPrefix(rootVersion, "v")
		if targetVersion == currentVersion {
			fmt.Printf("Already at version %s\n", rootVersion)
			return nil
		}

		osName := mapGoOS(runtime.GOOS)
		archName := mapGoArch(runtime.GOARCH)
		if osName == "" || archName == "" {
			return fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
		}

		archiveName := fmt.Sprintf("bookleaf_%s_%s.tar.gz", osName, archName)
		downloadURL := fmt.Sprintf(
			"https://github.com/%s/%s/releases/download/v%s/%s",
			ghOwner, ghRepo, targetVersion, archiveName,
		)

		fmt.Printf("Downloading bookleaf v%s...\n", targetVersion)
		resp, err := http.Get(downloadURL)
		if err != nil {
			return fmt.Errorf("download: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
		}

		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return fmt.Errorf("decompress: %w", err)
		}
		defer gzReader.Close()

		tarReader := tar.NewReader(gzReader)
		var binaryData []byte
		for {
			header, err := tarReader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("read archive: %w", err)
			}
			if header.Typeflag == tar.TypeReg && strings.HasSuffix(header.Name, "/"+binaryName) {
				binaryData, err = io.ReadAll(tarReader)
				if err != nil {
					return fmt.Errorf("read binary from archive: %w", err)
				}
				break
			}
		}

		if binaryData == nil {
			return fmt.Errorf("binary not found in archive")
		}

		binaryPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("find binary path: %w", err)
		}
		binaryPath, err = filepath.EvalSymlinks(binaryPath)
		if err != nil {
			return fmt.Errorf("resolve binary path: %w", err)
		}

		tmpFile := filepath.Join(os.TempDir(), "bookleaf-update")
		if err := os.WriteFile(tmpFile, binaryData, 0755); err != nil {
			return fmt.Errorf("write temp binary: %w", err)
		}

		if err := os.Rename(tmpFile, binaryPath); err != nil {
			if os.IsPermission(err) {
				fmt.Println("Need sudo to update the binary...")
				cmd := exec.Command("sudo", "mv", tmpFile, binaryPath)
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					os.Remove(tmpFile)
					return fmt.Errorf("sudo mv binary: %w", err)
				}
			} else {
				os.Remove(tmpFile)
				return fmt.Errorf("replace binary: %w", err)
			}
		}

		format.PrintSuccess(fmt.Sprintf("Updated to v%s", targetVersion))
		return nil
	},
}

func mapGoOS(goos string) string {
	switch goos {
	case "linux":
		return "Linux"
	case "darwin":
		return "Darwin"
	case "windows":
		return "Windows"
	default:
		return ""
	}
}

func mapGoArch(arch string) string {
	switch arch {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "arm64"
	default:
		return ""
	}
}

func fetchLatestVersion() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", ghOwner, ghRepo)
	resp, err := http.Get(url)
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		var release struct {
			TagName string `json:"tag_name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&release); err == nil && release.TagName != "" {
			return strings.TrimPrefix(release.TagName, "v"), nil
		}
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://github.com/%s/%s/releases/latest", ghOwner, ghRepo), nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err = client.Do(req)
	if err != nil {
		return "", fmt.Errorf("resolve latest version: %w", err)
	}
	defer resp.Body.Close()

	location := resp.Header.Get("Location")
	if location == "" {
		return "", fmt.Errorf("could not determine latest version")
	}

	parts := strings.Split(location, "/tag/")
	if len(parts) < 2 {
		return "", fmt.Errorf("could not parse version from: %s", location)
	}

	return strings.TrimPrefix(parts[len(parts)-1], "v"), nil
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
