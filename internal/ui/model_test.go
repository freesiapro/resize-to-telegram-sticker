package ui

import "testing"

func TestNewModelDefaults(t *testing.T) {
	m := NewModel()
	if m.state != "select" {
		t.Fatalf("expected state=select, got=%s", m.state)
	}
	if len(m.modeList.Items()) != 2 {
		t.Fatalf("expected 2 mode items, got=%d", len(m.modeList.Items()))
	}
	if m.pathInput.Placeholder != "/path/to/file/or/dir" {
		t.Fatalf("unexpected placeholder: %s", m.pathInput.Placeholder)
	}
}
