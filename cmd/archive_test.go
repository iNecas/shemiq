package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestArchiveMovesTaskWithoutChangingContents(t *testing.T) {
	project := t.TempDir()
	tasks := filepath.Join(project, ".shemiq", "tasks")
	first := filepath.Join(tasks, "first")
	if err := os.MkdirAll(filepath.Join(first, "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"top-level.md":   "unvalidated metadata stays untouched\n",
		"nested/note.md": "subtask with arbitrary text\n",
		"data.bin":       "\x00\xff\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(first, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	second := filepath.Join(tasks, "second")
	if err := os.Mkdir(second, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(second, "notes.txt"), []byte("no top-level file"), 0644); err != nil {
		t.Fatal(err)
	}
	cwd := filepath.Join(project, "src", "nested")
	if err := os.MkdirAll(cwd, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwd)

	for _, tc := range []struct {
		name, arg, source string
		files             map[string]string
	}{
		{"file path", filepath.Join("..", "..", ".shemiq", "tasks", "first", "top-level.md"), first, files},
		{"directory path", second, second, map[string]string{"notes.txt": "no top-level file"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := time.Now().UTC().Format("2006-01-02")
			stdout, stderr, err := runCommand("", "archive", tc.arg)
			after := time.Now().UTC().Format("2006-01-02")
			if err != nil || stderr != "" {
				t.Fatalf("archive: stdout=%q stderr=%q err=%v", stdout, stderr, err)
			}
			archive := filepath.Join(project, ".shemiq", "archive")
			path := strings.TrimSuffix(stdout, "\n")
			if stdout != path+"\n" || (path != filepath.Join(archive, before+"-"+filepath.Base(tc.source)) && path != filepath.Join(archive, after+"-"+filepath.Base(tc.source))) {
				t.Fatalf("stdout = %q, want UTC-dated absolute destination under %s", stdout, archive)
			}
			if _, err := os.Lstat(tc.source); !os.IsNotExist(err) {
				t.Fatalf("source still exists: %v", err)
			}
			for name, want := range tc.files {
				got, err := os.ReadFile(filepath.Join(path, name))
				if err != nil || string(got) != want {
					t.Fatalf("archived %s = %q, err=%v; want %q", name, got, err, want)
				}
			}
		})
	}
}

func TestArchiveRejectsCollisionAndInvalidPaths(t *testing.T) {
	project := t.TempDir()
	tasks := filepath.Join(project, ".shemiq", "tasks")
	source := filepath.Join(tasks, "keep")
	if err := os.MkdirAll(source, 0755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(source, "top-level.md")
	if err := os.WriteFile(file, []byte("unchanged"), 0644); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(project, ".shemiq", "archive", time.Now().UTC().Format("2006-01-02")+"-keep")
	if err := os.MkdirAll(destination, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "sentinel"), []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(project, "outside")
	if err := os.Mkdir(outside, 0755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(tasks, "linked")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	t.Chdir(project)
	for _, tc := range []struct{ name, path, message string }{
		{"collision", file, "already exists"},
		{"outside tasks", outside, "immediate child"},
		{"missing file", filepath.Join(source, "missing.md"), "no such file"},
		{"other file", filepath.Join(destination, "sentinel"), "not a task directory"},
		{"symlink", link, "not a task directory"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, err := runCommand("", "archive", tc.path)
			if err == nil || stdout != "" || !strings.Contains(stderr, tc.message) {
				t.Fatalf("archive %s: stdout=%q stderr=%q err=%v; want %q", tc.path, stdout, stderr, err, tc.message)
			}
			data, err := os.ReadFile(file)
			if err != nil || string(data) != "unchanged" {
				t.Fatalf("source changed: %q, %v", data, err)
			}
			if data, err := os.ReadFile(filepath.Join(destination, "sentinel")); err != nil || string(data) != "existing" {
				t.Fatalf("destination changed: %q, %v", data, err)
			}
		})
	}
}

func TestArchiveRequiresExistingProject(t *testing.T) {
	cwd := t.TempDir()
	t.Chdir(cwd)
	stdout, stderr, err := runCommand("", "archive", "some-task")
	if err == nil || stdout != "" || !strings.Contains(stderr, "no .shemiq directory found") {
		t.Fatalf("archive without project: stdout=%q stderr=%q err=%v", stdout, stderr, err)
	}
	if _, err := os.Lstat(filepath.Join(cwd, ".shemiq")); !os.IsNotExist(err) {
		t.Fatalf("archive created a project: %v", err)
	}
}
