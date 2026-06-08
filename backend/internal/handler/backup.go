package handler

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/knowledgeos/backend/internal/service"
)

type BackupHandler struct {
	svc *service.SnapshotService
}

func NewBackupHandler(svc *service.SnapshotService) *BackupHandler {
	return &BackupHandler{svc: svc}
}

// Snapshot builds a fresh tar.gz (dump.sql + code.tar.gz + metadata.json) and
// streams it to the caller. The intermediate files are removed afterwards.
func (h *BackupHandler) Snapshot(w http.ResponseWriter, r *http.Request) {
	archivePath, cleanup, err := h.svc.Build(r.Context())
	if err != nil {
		if errors.Is(err, service.ErrSnapshotInProgress) {
			Error(w, http.StatusConflict, err.Error())
			return
		}
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()

	f, err := os.Open(archivePath)
	if err != nil {
		Error(w, http.StatusInternalServerError, "could not open snapshot archive")
		return
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		Error(w, http.StatusInternalServerError, "could not stat snapshot archive")
		return
	}

	filename := fmt.Sprintf("snapshot-%s.tar.gz", time.Now().UTC().Format("20060102-150405"))
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	// ServeContent streams the file (supporting range requests) and sets
	// Content-Length from the ReadSeeker, keeping memory usage bounded.
	http.ServeContent(w, r, filename, stat.ModTime(), f)
}
