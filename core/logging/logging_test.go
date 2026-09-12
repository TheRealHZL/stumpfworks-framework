package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestNewJSONRedactsSensitiveAttributes(t *testing.T) {
	var output bytes.Buffer
	logger := NewJSON(&output, slog.LevelInfo)
	logger.Info("configured", "database_url", "postgres://user:secret@example/app", "component", "test")
	logged := output.String()
	if strings.Contains(logged, "postgres://") || strings.Contains(logged, "user:secret") {
		t.Fatalf("secret was logged: %s", logged)
	}
	if !strings.Contains(logged, redacted) || !strings.Contains(logged, `"component":"test"`) {
		t.Fatalf("unexpected log: %s", logged)
	}
}
