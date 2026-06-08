package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ComponentMeta struct {
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
	Files  int    `json:"files,omitempty"`
}

// Metadata mirrors the metadata.json produced by the main application's
// snapshot endpoint.
type Metadata struct {
	CreatedAt     time.Time                `json:"created_at"`
	GitCommit     string                   `json:"git_commit"`
	SchemaVersion string                   `json:"schema_version"`
	Components    map[string]ComponentMeta `json:"components"`
}

const snapshotEndpoint = "/api/v1/backup/snapshot"

// downloadSnapshot fetches the snapshot archive from the main application and
// streams it to dest.
func downloadSnapshot(cfg Config, client *http.Client, dest string) error {
	url := strings.TrimRight(cfg.AppURL, "/") + snapshotEndpoint
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+cfg.AppAPIKey)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request snapshot: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("snapshot endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create archive file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("write archive: %w", err)
	}
	return nil
}

// extractTarGz extracts a gzip-compressed tar archive into destDir, guarding
// against path traversal (zip-slip).
func extractTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip open %s: %w", archivePath, err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	cleanDest := filepath.Clean(destDir)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read: %w", err)
		}

		target := filepath.Join(cleanDest, hdr.Name)
		if target != cleanDest && !strings.HasPrefix(target, cleanDest+string(os.PathSeparator)) {
			return fmt.Errorf("unsafe path in archive: %q", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			// Preserve the original file mode (e.g. the executable bit) so the
			// round-trip into the git backup branch keeps permissions.
			mode := os.FileMode(hdr.Mode).Perm()
			if mode == 0 {
				mode = 0o644
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		default:
			// Skip symlinks and other special entries.
		}
	}
	return nil
}

func readMetadata(path string) (Metadata, error) {
	var m Metadata
	b, err := os.ReadFile(path)
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return m, fmt.Errorf("parse metadata.json: %w", err)
	}
	return m, nil
}

// verifyComponent checks that the file at path matches the size and checksum
// recorded in metadata.
func verifyComponent(path string, meta ComponentMeta) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Size() != meta.Bytes {
		return fmt.Errorf("%s size mismatch: got %d, expected %d", filepath.Base(path), info.Size(), meta.Bytes)
	}
	if meta.SHA256 == "" {
		return fmt.Errorf("%s: metadata is missing sha256", filepath.Base(path))
	}
	sum, err := sha256File(path)
	if err != nil {
		return err
	}
	if sum != meta.SHA256 {
		return fmt.Errorf("%s checksum mismatch", filepath.Base(path))
	}
	return nil
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
