package service

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	applog "github.com/knowledgeos/backend/internal/logger"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/knowledgeos/backend/internal/domain"
	"gorm.io/gorm"
)

// ErrSnapshotInProgress is returned when a snapshot build is already running.
// Concurrent calls are rejected rather than queued so we never run two pg_dump
// processes at once.
var ErrSnapshotInProgress = errors.New("snapshot already in progress")

// codeExcludeDirs are directory names skipped when building code.tar.gz so the
// snapshot carries sources only — no binaries, dependencies or VCS metadata.
var codeExcludeDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	"bin":          true,
	".idea":        true,
	".vscode":      true,
	"tmp":          true,
	".cache":       true,
}

// codeExcludeSuffixes are file extensions skipped in code.tar.gz: keys/certs and
// other secret material, build artifacts/binaries, logs and office docs.
var codeExcludeSuffixes = []string{
	".key", ".pem", ".pfx", ".p12", ".keystore", ".crt", ".cer", ".der", // keys / certs
	".jks", ".ppk", ".ovpn", ".kdbx", ".gpg", ".asc", // more secret material
	".exe", ".dll", ".so", ".dylib", ".test", ".o", ".a", ".out", // binaries / build artifacts
	".log", ".tsbuildinfo", ".docx", ".doc", ".xlsx",
}

// codeExcludeNames are exact base names of well-known credential files.
var codeExcludeNames = map[string]bool{
	".pgpass": true, ".netrc": true, ".npmrc": true, ".dockercfg": true,
	"credentials": true, "secrets.yml": true, "secrets.yaml": true, "secrets.json": true,
}

// shouldExcludeFile reports whether a regular file must be kept out of the code
// snapshot. CRITICAL: secret files (.env and friends, SSH/TLS keys) must never
// be packaged — the snapshot is pushed to a GitHub repository. The .env.example
// / .env.sample templates carry no secrets and are kept.
func shouldExcludeFile(name string) bool {
	if name == ".env" || (strings.HasPrefix(name, ".env.") && name != ".env.example" && name != ".env.sample") {
		return true
	}
	if codeExcludeNames[strings.ToLower(name)] {
		return true
	}
	switch {
	case name == ".DS_Store", strings.HasPrefix(name, "~$"):
		return true
	case strings.HasPrefix(name, "id_rsa"), strings.HasPrefix(name, "id_dsa"),
		strings.HasPrefix(name, "id_ecdsa"), strings.HasPrefix(name, "id_ed25519"):
		return true
	}
	lower := strings.ToLower(name)
	for _, suf := range codeExcludeSuffixes {
		if strings.HasSuffix(lower, suf) {
			return true
		}
	}
	return false
}

type SnapshotConfig struct {
	PGHost     string
	PGPort     int
	PGUser     string
	PGPassword string
	PGDatabase string

	CodePath   string
	GitCommit  string
	CommitFile string
}

type SnapshotService struct {
	cfg       SnapshotConfig
	db        *gorm.DB
	exporter  *ExportService
	companies domain.CompanyRepository
	mu        sync.Mutex
}

// NewSnapshotService executes the service.NewSnapshotService operation.
func NewSnapshotService(cfg SnapshotConfig, db *gorm.DB, exporter *ExportService, companies domain.CompanyRepository) *SnapshotService {
	return &SnapshotService{cfg: cfg, db: db, exporter: exporter, companies: companies}
}

type componentMeta struct {
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
	Files  int    `json:"files,omitempty"`
}

type snapshotMetadata struct {
	CreatedAt     time.Time                `json:"created_at"`
	GitCommit     string                   `json:"git_commit"`
	SchemaVersion string                   `json:"schema_version"`
	Components    map[string]componentMeta `json:"components"`
}

// Build produces a tar.gz archive containing dump.sql, code.tar.gz and
// metadata.json. It returns the path to the archive plus a cleanup function the
// caller must invoke once the archive has been streamed to the client. The
// pg_dump mutex is released as soon as the archive is built (before streaming).
func (s *SnapshotService) Build(ctx context.Context) (archivePath string, cleanup func(), err error) {
	applog.TraceCall(ctx, "service.SnapshotService.Build")
	if !s.mu.TryLock() {
		return "", nil, ErrSnapshotInProgress
	}

	workDir, err := os.MkdirTemp("", "kos-snapshot-*")
	if err != nil {
		s.mu.Unlock()
		return "", nil, fmt.Errorf("create temp dir: %w", err)
	}

	success := false
	defer func() {
		s.mu.Unlock()
		if !success {
			_ = os.RemoveAll(workDir)
		}
	}()

	dumpPath := filepath.Join(workDir, "dump.sql")
	if err := s.pgDump(ctx, dumpPath); err != nil {
		return "", nil, err
	}
	dumpBytes, dumpSum, err := fileStat(dumpPath)
	if err != nil {
		return "", nil, err
	}

	codePath := filepath.Join(workDir, "code.tar.gz")
	codeFiles, err := tarGzDir(s.cfg.CodePath, codePath)
	if err != nil {
		return "", nil, err
	}
	codeBytes, codeSum, err := fileStat(codePath)
	if err != nil {
		return "", nil, err
	}

	// Logical export (the same JSON the admin "Export" button produces, for every
	// company) — a portable, re-importable complement to the raw pg_dump.
	exportPath := filepath.Join(workDir, "export.json")
	exportBytes, exportSum, err := s.writeExport(ctx, exportPath)
	if err != nil {
		return "", nil, err
	}

	meta := snapshotMetadata{
		CreatedAt:     time.Now().UTC(),
		GitCommit:     s.resolveCommit(),
		SchemaVersion: s.schemaVersion(ctx),
		Components: map[string]componentMeta{
			"dump.sql":    {Bytes: dumpBytes, SHA256: dumpSum},
			"code.tar.gz": {Bytes: codeBytes, SHA256: codeSum, Files: codeFiles},
			"export.json": {Bytes: exportBytes, SHA256: exportSum},
		},
	}
	metaPath := filepath.Join(workDir, "metadata.json")
	if err := writeJSON(metaPath, meta); err != nil {
		return "", nil, err
	}

	archivePath = filepath.Join(workDir, "snapshot.tar.gz")
	if err := buildArchive(archivePath, []string{metaPath, dumpPath, codePath, exportPath}); err != nil {
		return "", nil, err
	}

	success = true
	cleanup = func() { _ = os.RemoveAll(workDir) }
	return archivePath, cleanup, nil
}

func (s *SnapshotService) pgDump(ctx context.Context, outPath string) error {
	applog.TraceCall(ctx, "service.SnapshotService.pgDump")
	args := []string{
		"--serializable-deferrable",
		"--no-owner",
		"--no-privileges",
		"-h", s.cfg.PGHost,
		"-p", fmt.Sprintf("%d", s.cfg.PGPort),
		"-U", s.cfg.PGUser,
		"-d", s.cfg.PGDatabase,
		"-f", outPath,
	}
	cmd := exec.CommandContext(ctx, "pg_dump", args...)
	cmd.Env = append(os.Environ(), "PGPASSWORD="+s.cfg.PGPassword)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_dump failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// companyExport pairs a company with the logical export of its knowledge base.
type companyExport struct {
	CompanyID   string      `json:"company_id"`
	CompanyName string      `json:"company_name"`
	Data        *ExportData `json:"data"`
}

// exportBundle is the export.json payload: the logical export of every company,
// each Data block identical to what GET /api/v1/export returns for that company.
type exportBundle struct {
	ExportedAt time.Time       `json:"exported_at"`
	Companies  []companyExport `json:"companies"`
}

// writeExport produces export.json (the logical export of all companies) and
// returns its size and sha256. When the exporter is not wired it writes an
// empty bundle so the component is always present and verifiable.
func (s *SnapshotService) writeExport(ctx context.Context, outPath string) (int64, string, error) {
	applog.TraceCall(ctx, "service.SnapshotService.writeExport")
	bundle := exportBundle{ExportedAt: time.Now().UTC(), Companies: []companyExport{}}
	if s.exporter != nil && s.companies != nil {
		companies, _, err := s.companies.List(ctx, domain.CompanyFilter{Page: 1, Limit: 10000})
		if err != nil {
			return 0, "", fmt.Errorf("list companies for export: %w", err)
		}
		for _, c := range companies {
			data, err := s.exporter.Export(ctx, c.ID)
			if err != nil {
				return 0, "", fmt.Errorf("export company %s: %w", c.ID, err)
			}
			bundle.Companies = append(bundle.Companies, companyExport{
				CompanyID:   c.ID.String(),
				CompanyName: c.Name,
				Data:        data,
			})
		}
	}
	if err := writeJSON(outPath, bundle); err != nil {
		return 0, "", err
	}
	return fileStat(outPath)
}

func (s *SnapshotService) resolveCommit() string {
	if s.cfg.GitCommit != "" {
		return s.cfg.GitCommit
	}
	if s.cfg.CommitFile != "" {
		if b, err := os.ReadFile(s.cfg.CommitFile); err == nil {
			if v := strings.TrimSpace(string(b)); v != "" {
				return v
			}
		}
	}
	return "unknown"
}

func (s *SnapshotService) schemaVersion(ctx context.Context) string {
	applog.TraceCall(ctx, "service.SnapshotService.schemaVersion")
	var filename string
	err := s.db.WithContext(ctx).
		Raw("SELECT filename FROM schema_migrations ORDER BY filename DESC LIMIT 1").
		Scan(&filename).Error
	if err != nil || filename == "" {
		return "unknown"
	}
	return filename
}

// tarGzDir writes a gzip-compressed tar of srcDir to outPath, skipping excluded
// directories. It returns the number of regular files written. A missing source
// directory yields a valid but empty archive so the endpoint stays available.
func tarGzDir(srcDir, outPath string) (int, error) {
	out, err := os.Create(outPath)
	if err != nil {
		return 0, fmt.Errorf("create code archive: %w", err)
	}
	defer out.Close()

	gz := gzip.NewWriter(out)
	tw := tar.NewWriter(gz)

	fileCount := 0
	info, statErr := os.Stat(srcDir)
	if statErr == nil && info.IsDir() {
		err = filepath.Walk(srcDir, func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(srcDir, path)
			if err != nil {
				return err
			}
			if rel == "." {
				return nil
			}
			if fi.IsDir() {
				if codeExcludeDirs[fi.Name()] {
					return filepath.SkipDir
				}
				return nil
			}
			// Only regular files; skip symlinks, sockets, devices.
			if !fi.Mode().IsRegular() {
				return nil
			}
			// Never package secrets or build artifacts.
			if shouldExcludeFile(fi.Name()) {
				return nil
			}
			hdr, err := tar.FileInfoHeader(fi, "")
			if err != nil {
				return err
			}
			hdr.Name = filepath.ToSlash(rel)
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			if _, err := io.Copy(tw, f); err != nil {
				return err
			}
			fileCount++
			return nil
		})
	}

	if cerr := tw.Close(); cerr != nil && err == nil {
		err = cerr
	}
	if cerr := gz.Close(); cerr != nil && err == nil {
		err = cerr
	}
	if err != nil {
		return 0, fmt.Errorf("tar code: %w", err)
	}
	return fileCount, nil
}

// buildArchive packs the given files (by their base names) into a tar.gz at outPath.
func buildArchive(outPath string, files []string) error {
	out, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create archive: %w", err)
	}
	defer out.Close()

	gz := gzip.NewWriter(out)
	tw := tar.NewWriter(gz)

	for _, f := range files {
		if err := addFileToTar(tw, f, filepath.Base(f)); err != nil {
			tw.Close()
			gz.Close()
			return err
		}
	}

	if err := tw.Close(); err != nil {
		gz.Close()
		return err
	}
	return gz.Close()
}

func addFileToTar(tw *tar.Writer, path, name string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	hdr, err := tar.FileInfoHeader(fi, "")
	if err != nil {
		return err
	}
	hdr.Name = name
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(tw, f)
	return err
}

func fileStat(path string) (int64, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, "", err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return 0, "", err
	}
	return n, hex.EncodeToString(h.Sum(nil)), nil
}

func writeJSON(path string, v interface{}) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}
