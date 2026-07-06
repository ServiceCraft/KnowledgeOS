package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// IndexEntry is one record in $BACKUPS_PATH/index.json.
type IndexEntry struct {
	Timestamp        string    `json:"timestamp"`
	CreatedAt        time.Time `json:"created_at"`
	GitCommit        string    `json:"git_commit"`
	SchemaVersion    string    `json:"schema_version"`
	Tag              string    `json:"tag"`
	DumpFile         string    `json:"dump_file"`
	DumpBytes        int64     `json:"dump_bytes"`         // compressed size
	Checksum         string    `json:"checksum_sha256"`    // of dump.sql.gz
	DumpSourceSHA256 string    `json:"dump_source_sha256"` // of the uncompressed dump.sql

	// Logical export bundle (export.json) stored gzipped alongside the dump.
	// Empty when the snapshot carries no export (older app versions).
	ExportFile     string `json:"export_file,omitempty"`
	ExportBytes    int64  `json:"export_bytes,omitempty"`           // compressed size
	ExportChecksum string `json:"export_checksum_sha256,omitempty"` // of export.json.gz
}

func indexPath(cfg Config) string { return filepath.Join(cfg.BackupsPath, "index.json") }

// storeDump writes the gzipped dump, metadata and checksum into a per-timestamp
// directory under BACKUPS_PATH and returns the index entry describing it.
func storeDump(cfg Config, extractedDir, timestamp, tag string, meta Metadata) (IndexEntry, error) {
	dir := filepath.Join(cfg.BackupsPath, timestamp)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return IndexEntry{}, fmt.Errorf("create snapshot dir: %w", err)
	}

	dumpGz := filepath.Join(dir, "dump.sql.gz")
	if err := gzipFile(filepath.Join(extractedDir, "dump.sql"), dumpGz); err != nil {
		_ = os.RemoveAll(dir)
		return IndexEntry{}, fmt.Errorf("gzip dump: %w", err)
	}

	if err := copyFile(filepath.Join(extractedDir, "metadata.json"), filepath.Join(dir, "metadata.json"), 0o600); err != nil {
		_ = os.RemoveAll(dir)
		return IndexEntry{}, fmt.Errorf("copy metadata: %w", err)
	}

	sum, err := sha256File(dumpGz)
	if err != nil {
		_ = os.RemoveAll(dir)
		return IndexEntry{}, err
	}

	info, err := os.Stat(dumpGz)
	if err != nil {
		_ = os.RemoveAll(dir)
		return IndexEntry{}, err
	}

	// Optional logical export bundle: store gzipped beside the dump.
	checksums := sum + "  dump.sql.gz\n"
	var exportFile, exportSum string
	var exportBytes int64
	if _, serr := os.Stat(filepath.Join(extractedDir, "export.json")); serr == nil {
		exportGz := filepath.Join(dir, "export.json.gz")
		if err := gzipFile(filepath.Join(extractedDir, "export.json"), exportGz); err != nil {
			_ = os.RemoveAll(dir)
			return IndexEntry{}, fmt.Errorf("gzip export: %w", err)
		}
		exportSum, err = sha256File(exportGz)
		if err != nil {
			_ = os.RemoveAll(dir)
			return IndexEntry{}, err
		}
		einfo, err := os.Stat(exportGz)
		if err != nil {
			_ = os.RemoveAll(dir)
			return IndexEntry{}, err
		}
		exportFile = "export.json.gz"
		exportBytes = einfo.Size()
		checksums += exportSum + "  export.json.gz\n"
	}

	if err := os.WriteFile(filepath.Join(dir, "checksum.sha256"), []byte(checksums), 0o600); err != nil {
		_ = os.RemoveAll(dir)
		return IndexEntry{}, err
	}

	return IndexEntry{
		Timestamp:        timestamp,
		CreatedAt:        meta.CreatedAt,
		GitCommit:        meta.GitCommit,
		SchemaVersion:    meta.SchemaVersion,
		Tag:              tag,
		DumpFile:         "dump.sql.gz",
		DumpBytes:        info.Size(),
		Checksum:         sum,
		DumpSourceSHA256: meta.Components["dump.sql"].SHA256,
		ExportFile:       exportFile,
		ExportBytes:      exportBytes,
		ExportChecksum:   exportSum,
	}, nil
}

func loadIndex(cfg Config) ([]IndexEntry, error) {
	b, err := os.ReadFile(indexPath(cfg))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entries []IndexEntry
	if err := json.Unmarshal(b, &entries); err != nil {
		return nil, fmt.Errorf("parse index.json: %w", err)
	}
	return entries, nil
}

func saveIndex(cfg Config, entries []IndexEntry) error {
	b, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(indexPath(cfg), b, 0o600)
}

// appendAndRotate records the new entry then enforces the FIFO retention limit,
// deleting the oldest snapshots beyond BACKUPS_KEEP. Returns how many were
// removed.
func appendAndRotate(cfg Config, entry IndexEntry) (int, error) {
	entries, err := loadIndex(cfg)
	if err != nil {
		return 0, err
	}
	entries = append(entries, entry)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Timestamp < entries[j].Timestamp })

	var drop []IndexEntry
	if len(entries) > cfg.BackupsKeep {
		drop = entries[:len(entries)-cfg.BackupsKeep]
		entries = entries[len(entries)-cfg.BackupsKeep:]
	}

	// Persist the trimmed index FIRST, then delete directories. A crash between
	// the two leaves harmless orphan dirs rather than an index that references
	// deleted dumps and omits the just-stored one.
	if err := saveIndex(cfg, entries); err != nil {
		return 0, err
	}

	removed := 0
	for _, e := range drop {
		if err := os.RemoveAll(filepath.Join(cfg.BackupsPath, e.Timestamp)); err != nil {
			return removed, fmt.Errorf("rotate %s: %w", e.Timestamp, err)
		}
		removed++
	}
	return removed, nil
}

func gzipFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	gz := gzip.NewWriter(out)
	if _, err := io.Copy(gz, in); err != nil {
		gz.Close()
		return err
	}
	return gz.Close()
}
