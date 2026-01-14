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

func TestBrowseViewPaneBordersMatchHeight(t *testing.T) {
	m := NewModelWithDeps("/tmp", fakeDirLister{}, fakeExpander{})
	m.width = 80
	m.height = 24
	m.resizeLists()

	view := m.viewBrowse()
	lines := strings.Split(view, "\n")
	if len(lines) != m.height {
		empty := 0
		for _, line := range lines {
			if strings.TrimSpace(ansi.Strip(line)) == "" {
				empty++
			}
		}
		maxLines := 10
		if len(lines) < maxLines {
			maxLines = len(lines)
		}
		for i := 0; i < maxLines; i++ {
			t.Logf("line %d: %q", i, ansi.Strip(lines[i]))
		}
		t.Fatalf("expected %d lines, got %d (empty=%d)", m.height, len(lines), empty)
	}

	contentWidth, contentHeight := contentSize(m.width, m.height)
	layout := calcPaneLayout(contentWidth, contentHeight)
	if layout.borderSize == 0 {
		t.Fatal("expected borders enabled")
	}

	start := outerPadY
	end := outerPadY + layout.contentHeight
	leftStart := outerPadX
	leftEnd := outerPadX + layout.leftWidth - 1
	rightStart := outerPadX + layout.leftWidth + layout.dividerWidth
	rightEnd := rightStart + layout.rightWidth - 1
	dividerIndex := outerPadX + layout.leftWidth

	for i := start; i < end; i++ {
		line := []rune(ansi.Strip(lines[i]))
		if len(line) < m.width {
			t.Fatalf("line %d too short: %d", i, len(line))
		}
		if line[leftStart] == ' ' || line[leftEnd] == ' ' {
			t.Fatalf("left pane border missing at line %d", i)
		}
		if layout.dividerWidth > 0 && line[dividerIndex] == ' ' {
			t.Fatalf("divider missing at line %d", i)
		}
		if line[rightStart] == ' ' || line[rightEnd] == ' ' {
			t.Fatalf("right pane border missing at line %d", i)
		}
	}
}

func TestBrowseViewStatusAndHelpSections(t *testing.T) {
	m := NewModelWithDeps("/tmp", fakeDirLister{}, fakeExpander{})
	m.width = 80
	m.height = 24
	m.status = "Ready"
	m.resizeLists()

	view := m.viewBrowse()
	lines := strings.Split(view, "\n")
	if len(lines) != m.height {
		t.Fatalf("expected %d lines, got %d", m.height, len(lines))
	}

	contentWidth, contentHeight := contentSize(m.width, m.height)
	layout := calcPaneLayout(contentWidth, contentHeight)
	statusHeight := contentHeight - layout.contentHeight
	if statusHeight < 3 {
		t.Fatalf("expected status height >= 3, got %d", statusHeight)
	}

	statusStart := outerPadY + layout.contentHeight
	statusLines := lines[statusStart : statusStart+statusHeight]
	statusText := ansi.Strip(statusLines[0])
	dividerText := strings.TrimSpace(ansi.Strip(statusLines[1]))
	helpText := ansi.Strip(statusLines[statusHeight-1])

	if !strings.Contains(statusText, "Ready") {
		t.Fatalf("expected status line to include Ready, got %q", statusText)
	}
	if strings.Trim(dividerText, "-") != "" {
		t.Fatalf("expected divider line, got %q", dividerText)
	}
	if !strings.Contains(helpText, "Tab") || !strings.Contains(helpText, "Switch") {
		t.Fatalf("expected help line to include Tab Switch, got %q", helpText)
	}
}
