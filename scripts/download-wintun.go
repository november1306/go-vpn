package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	wintunURL = "https://www.wintun.net/builds/wintun-0.14.1.zip"
	tempFile  = "wintun.zip"
	// SHA256 checksum for wintun-0.14.1.zip (verified from official source)
	wintunSHA256   = "07c256185d6ee3652e09fa55c0b673e2624b565e02c4b9091c79ca7d2f24ef51"
	requestTimeout = 30 * time.Second
	maxRedirects   = 3
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
	// Create HTTP client with security restrictions
	client := &http.Client{
		Timeout: requestTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("too many redirects (max %d)", maxRedirects)
			}
			return nil
		},
	}

	resp, err := client.Get(url)
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

	// Copy data while calculating SHA256
	hash := sha256.New()
	multiWriter := io.MultiWriter(out, hash)
	_, err = io.Copy(multiWriter, resp.Body)
	if err != nil {
		return err
	}

	// Verify SHA256 checksum
	calculatedHash := hex.EncodeToString(hash.Sum(nil))
	if calculatedHash != wintunSHA256 {
		// Clean up invalid file
		os.Remove(filename)
		return fmt.Errorf("SHA256 checksum mismatch:\n  expected: %s\n  got:      %s", wintunSHA256, calculatedHash)
	}

	fmt.Println("✓ SHA256 checksum verified")
	return nil
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
		defer rc.Close() // Fix resource leak - close immediately after open

		// Create destination file
		outFile, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer outFile.Close() // Ensure file is closed on any exit path

		// Copy content
		_, err = io.Copy(outFile, rc)
		if err != nil {
			return err
		}
	}

	return nil
}
