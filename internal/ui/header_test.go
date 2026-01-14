package ui

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestLeftHeaderLineTruncatesToWidth(t *testing.T) {
	line := leftHeaderLine("/very/long/path/that/should/truncate", "filter", 12)
	if ansi.StringWidth(line) > 12 {
		t.Fatalf("expected width <= 12, got %d: %q", ansi.StringWidth(line), line)
	}
}
