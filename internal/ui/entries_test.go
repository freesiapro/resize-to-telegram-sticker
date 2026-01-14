package ui

import "testing"

func TestFilterEntries(t *testing.T) {
	entries := []entryItem{
		{name: "cats", isDir: true},
		{name: "cat01.gif", isDir: false},
		{name: "dogs", isDir: true},
	}
	filtered := filterEntries(entries, "cat")
	if len(filtered) != 2 {
		t.Fatalf("expected 2, got %d", len(filtered))
	}
}
