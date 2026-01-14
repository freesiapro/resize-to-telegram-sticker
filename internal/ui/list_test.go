package ui

import "testing"

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
