package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestLeftHeaderLineTruncatesToWidth(t *testing.T) {
	line := leftHeaderLine("filter", 12)
	if ansi.StringWidth(line) > 12 {
		t.Fatalf("expected width <= 12, got %d: %q", ansi.StringWidth(line), line)
	}
}

func TestSearchHeaderNoEllipsisWhenEmpty(t *testing.T) {
	m := NewModelWithDeps("/tmp", fakeDirLister{}, fakeExpander{})
	m.width = 80
	m.height = 24
	m.resizeLists()

	contentWidth, contentHeight := contentSize(m.width, m.height)
	layout := calcPaneLayout(contentWidth, contentHeight)
	headerWidth := headerContentWidth(layout.leftInnerWidth)

	line := leftHeaderLine(m.filterInput.View(), headerWidth)
	if strings.Contains(line, "...") {
		t.Fatalf("expected no ellipsis, got %q", line)
	}
}
