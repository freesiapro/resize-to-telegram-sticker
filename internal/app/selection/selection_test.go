package selection

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandSelections(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "cats"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "cats", "a.gif"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	selections := []SelectionItem{
		{Path: filepath.Join(root, "cats"), IsDir: true},
		{Path: filepath.Join(root, "b.png"), IsDir: false},
	}
	if err := os.WriteFile(filepath.Join(root, "b.png"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	expander := SelectionExpander{}
	result, err := expander.Expand(selections, filepath.Join(root, "output"))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if result.DirCount != 1 || result.FileCount != 1 {
		t.Fatalf("unexpected counts: %+v", result)
	}
	if result.TotalFiles != 2 {
		t.Fatalf("unexpected total files: %+v", result)
	}

	var hasOutputDir bool
	for _, job := range result.Jobs {
		if job.OutputDir == filepath.Join(root, "output") {
			hasOutputDir = true
		}
	}
	if !hasOutputDir {
		t.Fatalf("unexpected output dirs: %+v", result.Jobs)
	}
}
