package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const (
	wintunURL = "https://www.wintun.net/builds/wintun-0.14.1.zip"
	tempFile  = "wintun.zip"
)

var archMap = map[string]string{
	"wintun/bin/amd64/wintun.dll": "lib/amd64/wintun.dll",
	"wintun/bin/arm64/wintun.dll": "lib/arm64/wintun.dll",
	"wintun/bin/arm/wintun.dll":   "lib/arm/wintun.dll",
	"wintun/bin/x86/wintun.dll":   "lib/x86/wintun.dll",
}

func main() {
	fmt.Println("Downloading WinTUN drivers...")

	// Create lib directories
	for _, destPath := range archMap {
		dir := filepath.Dir(destPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("Error creating directory %s: %v\n", dir, err)
			os.Exit(1)
		}
	}

	// Download ZIP file
	fmt.Printf("Downloading from %s...\n", wintunURL)
	if err := downloadFile(wintunURL, tempFile); err != nil {
		fmt.Printf("Error downloading: %v\n", err)
		os.Exit(1)
	}

	// Extract required DLL files
	fmt.Println("Extracting DLL files...")
	if err := extractDLLs(tempFile); err != nil {
		fmt.Printf("Error extracting: %v\n", err)
		os.Exit(1)
	}

	// Clean up
	os.Remove(tempFile)
	fmt.Println("WinTUN drivers downloaded successfully!")
}

func downloadFile(url, filename string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func extractDLLs(zipFile string) error {
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		// Check if this is one of the DLL files we need
		destPath, needed := archMap[f.Name]
		if !needed {
			continue
		}

		fmt.Printf("Extracting %s -> %s\n", f.Name, destPath)

		// Open source file in ZIP
		rc, err := f.Open()
		if err != nil {
			return err
		}

		// Create destination file
		outFile, err := os.Create(destPath)
		if err != nil {
			rc.Close()
			return err
		}

		// Copy content
		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}
