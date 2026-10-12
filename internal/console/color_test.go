package console

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestColorStatusValues(t *testing.T) {
	tests := []struct {
		status string
		code   string
	}{
		{"new", ansiCyan},
		{"refined", ansiYellow},
		{"done", ansiGreen},
	}
	for _, tt := range tests {
		result := ColorStatus(tt.status, true)
		if !strings.HasPrefix(result, tt.code) {
			t.Errorf("ColorStatus(%q, true) should start with %q, got %q",
				tt.status, tt.code, result)
		}
		if !strings.HasSuffix(result, ansiReset) {
			t.Errorf("ColorStatus(%q, true) should end with reset, got %q",
				tt.status, result)
		}
		if strings.Contains(result[len(tt.code):len(result)-len(ansiReset)], "\033") {
			t.Errorf("ColorStatus(%q, true) has extra escapes in status word", tt.status)
		}
	}

	// Color disabled
	for _, tt := range tests {
		result := ColorStatus(tt.status, false)
		if result != tt.status {
			t.Errorf("ColorStatus(%q, false) = %q, want %q", tt.status, result, tt.status)
		}
	}
}

func TestColorEnabledBufferedWriter(t *testing.T) {
	var buf bytes.Buffer
	if ColorEnabled(&buf) {
		t.Error("ColorEnabled should return false for bytes.Buffer")
	}
}

func TestColorEnabledNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	// Even if we had a terminal, NO_COLOR disables color.
	if ColorEnabled(os.Stdout) {
		t.Error("ColorEnabled should return false when NO_COLOR is set")
	}
}

func TestColorEnabledTermDumb(t *testing.T) {
	t.Setenv("TERM", "dumb")
	if ColorEnabled(os.Stdout) {
		t.Error("ColorEnabled should return false when TERM=dumb")
	}
}
