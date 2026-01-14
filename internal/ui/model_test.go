package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/resize-to-telegram-sticker/internal/app"
	"github.com/resize-to-telegram-sticker/internal/infra"
)

type fakeDirLister struct {
	entries []infra.DirEntry
	err     error
}

func (f fakeDirLister) ListDirEntries(_ string) ([]infra.DirEntry, error) {
	return f.entries, f.err
}

type fakeExpander struct {
	result app.ExpandResult
	err    error
}

func (f fakeExpander) Expand(_ []app.SelectionItem, _ string) (app.ExpandResult, error) {
	return f.result, f.err
}

func TestNewModelDefaults(t *testing.T) {
	m := NewModelWithDeps("/tmp", fakeDirLister{}, fakeExpander{})
	if m.state != stateBrowse {
		t.Fatalf("expected state=browse, got=%v", m.state)
	}
	if m.focus != focusLeft {
		t.Fatalf("expected focusLeft, got=%v", m.focus)
	}
	if m.outputDir != "./output" {
		t.Fatalf("expected outputDir ./output, got=%s", m.outputDir)
	}
	if m.filterInput.Value() != "" {
		t.Fatalf("expected empty filter, got=%s", m.filterInput.Value())
	}
}

func TestModelFilterInput(t *testing.T) {
	m := NewModelWithDeps("/tmp", fakeDirLister{}, fakeExpander{})
	m.focus = focusLeft
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = updated.(model)
	if m.filterInput.Value() != "c" {
		t.Fatalf("expected filter 'c', got '%s'", m.filterInput.Value())
	}
}

func TestLeftHeaderLineInlineSearch(t *testing.T) {
	line := leftHeaderLine("cat", 40)
	if strings.Contains(line, "\n") {
		t.Fatal("expected single line header")
	}
	if !strings.Contains(line, "Search: cat") {
		t.Fatalf("expected search text, got %q", line)
	}
	if strings.Contains(line, "Dir:") {
		t.Fatalf("expected no dir label, got %q", line)
	}
}

func TestConfirmRequiresSelection(t *testing.T) {
	m := NewModelWithDeps("/tmp", fakeDirLister{}, fakeExpander{})
	m.focus = focusRight
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	if m.err == nil {
		t.Fatal("expected error on empty selection")
	}
}

func TestConfirmDoesNotSkipWhenNoDirs(t *testing.T) {
	expander := fakeExpander{result: app.ExpandResult{DirCount: 0, FileCount: 1, TotalFiles: 1, OutputDirs: []string{"/tmp/output"}}}
	m := NewModelWithDeps("/tmp", fakeDirLister{}, expander)
	m.focus = focusRight
	m.selectedList = []app.SelectionItem{{Path: "/tmp/a.gif", IsDir: false}}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	if m.state != stateConfirm {
		t.Fatalf("expected stateConfirm, got=%v", m.state)
	}
	if cmd == nil {
		t.Fatal("expected cmd")
	}
	msg := cmd()
	updated, _ = m.Update(msg)
	m = updated.(model)
	if m.state != stateConfirm {
		t.Fatalf("expected confirm state, got=%v", m.state)
	}
}

func TestFilterInputHasBackground(t *testing.T) {
	m := NewModelWithDeps("/tmp", fakeDirLister{}, fakeExpander{})
	if _, ok := m.filterInput.TextStyle.GetBackground().(lipgloss.NoColor); ok {
		t.Fatal("expected filter input to have background color")
	}
	if _, ok := m.filterInput.PlaceholderStyle.GetBackground().(lipgloss.NoColor); ok {
		t.Fatal("expected filter placeholder to have background color")
	}
	if _, ok := m.filterInput.Cursor.TextStyle.GetBackground().(lipgloss.NoColor); ok {
		t.Fatal("expected filter cursor text style to have background color")
	}
}

func TestFilterInputWidthMatchesHeader(t *testing.T) {
	m := NewModelWithDeps("/tmp", fakeDirLister{}, fakeExpander{})
	m.width = 80
	m.height = 24
	m.resizeLists()

	contentWidth, contentHeight := contentSize(m.width, m.height)
	layout := calcPaneLayout(contentWidth, contentHeight)
	expected := layout.leftInnerWidth
	if expected < 1 {
		expected = 1
	}
	if m.filterInput.Width != expected {
		t.Fatalf("expected filter width %d, got %d", expected, m.filterInput.Width)
	}
}
