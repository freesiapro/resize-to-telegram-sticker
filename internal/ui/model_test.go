package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

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
	if m.filterText != "" {
		t.Fatalf("expected empty filter, got=%s", m.filterText)
	}
}

func TestModelFilterInput(t *testing.T) {
	m := NewModelWithDeps("/tmp", fakeDirLister{}, fakeExpander{})
	m.focus = focusLeft
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = updated.(model)
	if m.filterText != "c" {
		t.Fatalf("expected filter 'c', got '%s'", m.filterText)
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
