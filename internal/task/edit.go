package task

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// textEdit addresses a half-open byte range in the original document.
type textEdit struct {
	start, end int
	text       string
}

// applyEdits writes edits from the original snapshot to outPath without
// changing the snapshot. The caller owns staging, verification, and replacement.
func applyEdits(doc *parsedDocument, edits []textEdit, outPath string) error {
	if err := os.WriteFile(outPath, editedContent(doc, edits), 0644); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}
	return nil
}

// replaceEditedDocument copies verified content to a destination-local temporary
// file, preserves permissions, and renames it into place. This also works when
// the staged file is on another filesystem.
func replaceEditedDocument(path, staged string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	source, err := os.Open(staged)
	if err != nil {
		return err
	}
	defer source.Close()
	local, err := os.CreateTemp(filepath.Dir(path), ".shemiq-edit-*")
	if err != nil {
		return err
	}
	defer os.Remove(local.Name())
	defer local.Close()
	if _, err := io.Copy(local, source); err != nil {
		return err
	}
	if err := local.Chmod(info.Mode().Perm()); err != nil {
		return err
	}
	if err := local.Close(); err != nil {
		return err
	}
	return os.Rename(local.Name(), path)
}

// editedContent applies non-overlapping edits against original byte offsets
// without changing the document snapshot. It sorts edits in place.
func editedContent(doc *parsedDocument, edits []textEdit) []byte {
	sort.SliceStable(edits, func(i, j int) bool {
		return edits[i].start < edits[j].start
	})
	var output bytes.Buffer
	start := 0
	for _, edit := range edits {
		output.Write(doc.data[start:edit.start])
		output.WriteString(edit.text)
		start = edit.end
	}
	output.Write(doc.data[start:])
	return output.Bytes()
}

func documentLineEnding(doc *parsedDocument) string {
	if i := bytes.IndexByte(doc.data, '\n'); i >= 0 {
		if i > 0 && doc.data[i-1] == '\r' {
			return "\r\n"
		}
	}
	return "\n"
}

// ensureBlankLineBefore returns a prefix that ensures a
// blank line separator before the insertion point.
func ensureBlankLineBefore(data []byte, pos int, le string) string {
	if pos == 0 {
		return ""
	}
	// Count trailing newlines before pos.
	i := pos - 1
	newlines := 0
	for i >= 0 {
		if data[i] == '\n' {
			newlines++
			i--
			if i >= 0 && data[i] == '\r' {
				i--
			}
		} else {
			break
		}
	}
	switch {
	case newlines >= 2:
		return ""
	case newlines == 1:
		return le
	default:
		return le + le
	}
}
