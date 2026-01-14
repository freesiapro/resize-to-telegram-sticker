package ui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestStatusStylesNoBackground(t *testing.T) {
	styles := newStyles()
	if _, ok := styles.statusBar.GetBackground().(lipgloss.NoColor); !ok {
		t.Fatal("expected status bar without background color")
	}
	if _, ok := styles.helpBar.GetBackground().(lipgloss.NoColor); !ok {
		t.Fatal("expected help bar without background color")
	}
}
