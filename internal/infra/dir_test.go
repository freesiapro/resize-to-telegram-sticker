package infra

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListDirEntries(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.gif"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	entries, err := ListDirEntries(root)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	var hasDir, hasFile bool
	for _, e := range entries {
		if e.IsDir && e.Name == "sub" {
			hasDir = true
		}
		if !e.IsDir && e.Name == "a.gif" {
			hasFile = true
		}
	}
	if !hasDir || !hasFile {
		t.Fatalf("missing dir/file: %v", entries)
	}
}
