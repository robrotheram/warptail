package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v3"
)

// Release represents a GitHub release
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// Asset represents a GitHub release asset
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func CurrentExecutable() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return exePath, err
	}
	return filepath.Abs(exePath)
}

func fetchBinary(path string) {

	release, err := fetchLatestRelease()
	if err != nil {
		fmt.Printf("Error fetching latest release: %v\n", err)
		return
	}

	// Find the appropriate binary for the current OS and architecture
	var assetURL string
	if runtime.GOOS == "linux" {
		for _, asset := range release.Assets {
			if (runtime.GOARCH == "amd64" && strings.Contains(asset.Name, "amd64")) ||
				(runtime.GOARCH == "arm" && strings.Contains(asset.Name, "arm")) ||
				(runtime.GOARCH == "arm64" && strings.Contains(asset.Name, "arm64")) {
				assetURL = asset.BrowserDownloadURL
				break
			}
		}
	}

	if assetURL == "" {
		fmt.Println("No suitable binary found for the current OS and architecture")
		return
	}

	// Download the binary
	fmt.Println("Binary updated successfully to " + release.TagName)
	err = downloadBinary(assetURL, path)
	if err != nil {
		fmt.Printf("Error downloading binary: %v\n", err)
		return
	}

	// Make the binary executable
	err = os.Chmod(path, 0755)
	if err != nil {
		fmt.Printf("Error making binary executable: %v\n", err)
		return
	}

	fmt.Println("Binary updated successfully to " + release.TagName)
}

func Update(ctx context.Context, cmd *cli.Command) error {
	targetPath, _ := CurrentExecutable()
	oldPath := targetPath + ".old"
	newPath := targetPath + ".new"

	fetchBinary(newPath)

	_ = os.Remove(oldPath)
	err := os.Rename(targetPath, oldPath)
	if err != nil {
		return err
	}

	// move the new exectuable in to become the new program
	err = os.Rename(newPath, targetPath)

	if err != nil {
		// move unsuccessful
		rerr := os.Rename(oldPath, targetPath)
		if rerr != nil {
			return err
		}
		return err
	}
	return os.Remove(oldPath)
}

func fetchLatestRelease() (*Release, error) {
	url := "https://api.github.com/repos/robrotheram/warptail/releases/latest"
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch latest release: %s", resp.Status)
	}

	var release Release
	err = json.NewDecoder(resp.Body).Decode(&release)
	if err != nil {
		return nil, err
	}

	return &release, nil
}

func downloadBinary(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download binary: %s", resp.Status)
	}

	// Get the size of the file
	size := resp.ContentLength

	// Create a progress bar
	bar := progressbar.DefaultBytes(
		size,
		"downloading",
	)

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	// Create a multi-writer to write to both the file and the progress bar
	mw := io.MultiWriter(out, bar)

	_, err = io.Copy(mw, resp.Body)
	return err
}
