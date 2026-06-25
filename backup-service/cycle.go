package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// runCycle performs one full backup cycle (TZ §5.4): fetch snapshot, verify,
// push code to GitHub, store the data dump, rotate, and log the outcome. Code
// and data are handled independently so a GitHub outage does not discard a
// valid data snapshot; any partial failure makes the cycle return an error
// while keeping whatever succeeded.
func runCycle(cfg Config, log *Logger) error {
	log.Tracef("function called: %s", "main.runCycle")
	start := time.Now()
	timestamp := start.UTC().Format("20060102-150405")
	tag := "snapshot-" + timestamp
	log.Infof("cycle start (tag %s)", tag)

	if err := os.MkdirAll(cfg.BackupsPath, 0o700); err != nil {
		return fmt.Errorf("ensure backups path: %w", err)
	}

	// Work in a temp dir on the backups volume (ample space for large dumps).
	tmp, err := os.MkdirTemp(cfg.BackupsPath, ".work-")
	if err != nil {
		return fmt.Errorf("create work dir: %w", err)
	}
	defer os.RemoveAll(tmp)

	// --- Step 1: download + integrity check ---
	client := &http.Client{Timeout: cfg.HTTPTimeout}
	archive := filepath.Join(tmp, "snapshot.tar.gz")
	if err := downloadSnapshot(cfg, client, archive); err != nil {
		return fmt.Errorf("download: %w", err)
	}

	extracted := filepath.Join(tmp, "extracted")
	if err := os.MkdirAll(extracted, 0o755); err != nil {
		return err
	}
	if err := extractTarGz(archive, extracted); err != nil {
		return fmt.Errorf("extract snapshot: %w", err)
	}

	meta, err := readMetadata(filepath.Join(extracted, "metadata.json"))
	if err != nil {
		return fmt.Errorf("metadata: %w", err)
	}
	if err := verifyComponent(filepath.Join(extracted, "dump.sql"), meta.Components["dump.sql"]); err != nil {
		return fmt.Errorf("verify: %w", err)
	}
	if err := verifyComponent(filepath.Join(extracted, "code.tar.gz"), meta.Components["code.tar.gz"]); err != nil {
		return fmt.Errorf("verify: %w", err)
	}
	// export.json is optional (absent in snapshots from older app versions).
	if c, ok := meta.Components["export.json"]; ok {
		if err := verifyComponent(filepath.Join(extracted, "export.json"), c); err != nil {
			return fmt.Errorf("verify: %w", err)
		}
	}
	log.Infof("snapshot verified: commit=%s schema=%s dump=%d bytes code=%d bytes",
		meta.GitCommit, meta.SchemaVersion, meta.Components["dump.sql"].Bytes, meta.Components["code.tar.gz"].Bytes)

	// "No new data" notice (still stored — TZ §5.8).
	if prev := lastEntry(cfg); prev != nil && prev.DumpSourceSHA256 != "" &&
		prev.DumpSourceSHA256 == meta.Components["dump.sql"].SHA256 {
		log.Infof("no new data since previous snapshot %s (storing anyway)", prev.Timestamp)
	}

	// --- Step 2: code -> GitHub ---
	codeDir := filepath.Join(tmp, "code")
	if err := os.MkdirAll(codeDir, 0o755); err != nil {
		return err
	}
	var codeErr error
	if err := extractTarGz(filepath.Join(extracted, "code.tar.gz"), codeDir); err != nil {
		codeErr = fmt.Errorf("extract code: %w", err)
	} else if err := prepareRepo(cfg, log); err != nil {
		codeErr = err
	} else if err := commitAndPush(cfg, codeDir, tag, meta, log); err != nil {
		codeErr = err
	}
	if codeErr != nil {
		log.Errorf("code backup failed: %v", codeErr)
	}

	// --- Step 3 & 4: data -> registry + rotation ---
	var dataErr error
	entry, err := storeDump(cfg, extracted, timestamp, tag, meta)
	if err != nil {
		dataErr = err
		log.Errorf("data backup failed: %v", err)
	} else if removed, rerr := appendAndRotate(cfg, entry); rerr != nil {
		dataErr = fmt.Errorf("rotation: %w", rerr)
		log.Errorf("%v", dataErr)
	} else {
		if removed > 0 {
			log.Infof("rotation removed %d old snapshot(s), keeping %d", removed, cfg.BackupsKeep)
		}
		if entry.ExportFile != "" {
			log.Infof("stored data snapshot %s (dump %d bytes gz, export %d bytes gz)", timestamp, entry.DumpBytes, entry.ExportBytes)
		} else {
			log.Infof("stored data snapshot %s (%d bytes gz)", timestamp, entry.DumpBytes)
		}
	}

	// --- Step 5: outcome ---
	dur := time.Since(start).Round(time.Millisecond)
	if codeErr != nil || dataErr != nil {
		return fmt.Errorf("cycle finished with errors in %s (code: %v; data: %v)", dur, codeErr, dataErr)
	}
	log.Infof("cycle ok in %s (tag %s)", dur, tag)
	return nil
}

func lastEntry(cfg Config) *IndexEntry {
	entries, err := loadIndex(cfg)
	if err != nil || len(entries) == 0 {
		return nil
	}
	last := entries[0]
	for _, e := range entries[1:] {
		if e.Timestamp > last.Timestamp {
			last = e
		}
	}
	return &last
}
