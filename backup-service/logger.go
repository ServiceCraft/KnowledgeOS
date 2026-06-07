package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Logger writes structured lines to stdout and, when a journal path is set,
// appends them to a file-journal inside the backups directory (TZ §5.4 step 5).
type Logger struct {
	journalPath string
	secrets     []string // values to redact from any log line
}

func NewLogger(journalPath string, secrets ...string) *Logger {
	var filtered []string
	for _, s := range secrets {
		if s != "" {
			filtered = append(filtered, s)
		}
	}
	return &Logger{journalPath: journalPath, secrets: filtered}
}

func (l *Logger) redact(msg string) string {
	for _, s := range l.secrets {
		msg = strings.ReplaceAll(msg, s, "***")
	}
	return msg
}

func (l *Logger) write(level, format string, args ...interface{}) {
	line := fmt.Sprintf("%s [%s] %s",
		time.Now().UTC().Format(time.RFC3339), level, l.redact(fmt.Sprintf(format, args...)))
	fmt.Fprintln(os.Stdout, line)
	if l.journalPath != "" {
		if f, err := os.OpenFile(l.journalPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600); err == nil {
			fmt.Fprintln(f, line)
			_ = f.Close()
		}
	}
}

func (l *Logger) Infof(format string, args ...interface{})  { l.write("INFO", format, args...) }
func (l *Logger) Warnf(format string, args ...interface{})  { l.write("WARN", format, args...) }
func (l *Logger) Errorf(format string, args ...interface{}) { l.write("ERROR", format, args...) }

// JournalPath returns the canonical journal location inside the backups dir.
func JournalPath(backupsPath string) string {
	return filepath.Join(backupsPath, "backup.log")
}
