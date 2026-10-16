package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyEditsPreservesSnapshot(t *testing.T) {
	const original = "# Títle\r\nkeep \t\r\nold\r\nend"
	const edited = "# Títle\r\nkeep \t\r\nreplacement\r\nend!"

	for _, tc := range []struct {
		name       string
		outputFile string
		wantSource string
	}{
		{
			name:       "in place",
			outputFile: "example.md",
			wantSource: edited,
		},
		{
			name:       "separate output",
			outputFile: "edited.md",
			wantSource: original,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := storeFixture(t, dir, "example.md", original)
			if err := os.Chmod(path, 0640); err != nil {
				t.Fatal(err)
			}
			doc, err := parseMarkdownDocument(path, []byte(original))
			if err != nil {
				t.Fatal(err)
			}
			start := strings.Index(original, "old")
			// Deliberately unordered: offsets refer to the original snapshot,
			// even when an earlier replacement changes the content length.
			edits := []textEdit{
				{len(original), len(original), "!"},
				{start, start + len("old"), "replacement"},
			}
			outPath := filepath.Join(dir, tc.outputFile)
			if err := applyEdits(doc, edits, outPath); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(outPath)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != edited {
				t.Fatalf("output content = %q, want %q", data, edited)
			}
			data, err = os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != tc.wantSource {
				t.Fatalf("source content = %q, want %q", data, tc.wantSource)
			}
			if string(doc.data) != original {
				t.Fatal("editing changed the original snapshot")
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != 0640 {
				t.Fatalf("source permissions = %o, want 640", info.Mode().Perm())
			}
		})
	}
}
