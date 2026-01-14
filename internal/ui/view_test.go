package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestBrowseViewFillsWidth(t *testing.T) {
	m := NewModelWithDeps("/tmp", fakeDirLister{}, fakeExpander{})
	m.width = 80
	m.height = 24
	m.resizeLists()

	view := m.viewBrowse()
	lines := strings.Split(view, "\n")
	if len(lines) == 0 {
		t.Fatal("expected view output")
	}
	for i, line := range lines {
		if ansi.StringWidth(line) != m.width {
			t.Fatalf("line %d width=%d want=%d", i, ansi.StringWidth(line), m.width)
		}
	}
}
