package infra

import (
	"os"
	"path/filepath"
)

type DirEntry struct {
	Name  string
	Path  string
	IsDir bool
}

func ListDirEntries(root string) ([]DirEntry, error) {
	dirEntries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	entries := make([]DirEntry, 0, len(dirEntries))
	for _, e := range dirEntries {
		entries = append(entries, DirEntry{
			Name:  e.Name(),
			Path:  filepath.Join(root, e.Name()),
			IsDir: e.IsDir(),
		})
	}
	return entries, nil
}
