package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/x/ansi"
)

func TestNewListDelegateCompact(t *testing.T) {
	delegate := newListDelegate()
	if delegate.ShowDescription {
		t.Fatal("expected ShowDescription to be false")
	}
	if delegate.Spacing() != 0 {
		t.Fatalf("expected spacing 0, got %d", delegate.Spacing())
	}
}

func TestNewListModelCompact(t *testing.T) {
	listModel := newListModel()
	if listModel.ShowTitle() {
		t.Fatal("expected title hidden")
	}
	if listModel.ShowPagination() {
		t.Fatal("expected pagination hidden")
	}
	if listModel.ShowHelp() {
		t.Fatal("expected help hidden")
	}
	if listModel.ShowStatusBar() {
		t.Fatal("expected status bar hidden")
	}
	if listModel.ShowFilter() {
		t.Fatal("expected filter hidden")
	}
	if listModel.FilteringEnabled() {
		t.Fatal("expected filtering disabled")
	}
}

func TestSelectedRowFillsWidth(t *testing.T) {
	listModel := newListModel()
	listModel.SetSize(20, 5)
	listModel.SetItems([]list.Item{
		leftListItem{entry: entryItem{name: "sample.txt"}},
	})
	listModel.Select(0)

	view := listModel.View()
	lines := strings.Split(view, "\n")
	if len(lines) == 0 {
		t.Fatal("expected list view output")
	}
	if width := ansi.StringWidth(lines[0]); width != 20 {
		t.Fatalf("expected selected row width 20, got %d", width)
	}
}
