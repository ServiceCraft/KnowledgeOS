package logger

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestTraceErrNil(t *testing.T) {
	if err := TraceErr(context.Background(), "test", nil); err != nil {
		t.Fatalf("TraceErr(nil) = %v, want nil", err)
	}
}

func TestTraceErrLogsCallerAndMessage(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	ctx := logger.WithContext(context.Background())

	testErr := errors.New("boom")
	if got := TraceErr(ctx, "operation failed", testErr); got != testErr {
		t.Fatalf("TraceErr() = %v, want original error", got)
	}

	out := buf.String()
	if !strings.Contains(out, "boom") {
		t.Fatalf("log missing error: %s", out)
	}
	if !strings.Contains(out, "operation failed") {
		t.Fatalf("log missing message: %s", out)
	}
	if !strings.Contains(out, "logger_test.go") {
		t.Fatalf("log missing caller: %s", out)
	}
}
