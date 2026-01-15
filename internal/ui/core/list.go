package core

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type fullRowDelegate struct {
	list.DefaultDelegate
}

func newListDelegate() *fullRowDelegate {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetSpacing(0)

	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.
		Foreground(lipgloss.Color("250")).
		Padding(0, 0, 0, 1)
	delegate.Styles.DimmedTitle = delegate.Styles.DimmedTitle.
		Foreground(lipgloss.Color("242")).
		Padding(0, 0, 0, 1)
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("69")).
		Background(lipgloss.Color("237")).
		Bold(true).
		Padding(0, 0, 0, 1)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedTitle
	delegate.Styles.FilterMatch = delegate.Styles.FilterMatch.Foreground(lipgloss.Color("69"))

	return &fullRowDelegate{DefaultDelegate: delegate}
}

func (d *fullRowDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var (
		title, desc  string
		matchedRunes []int
		s            = &d.Styles
	)

	i, ok := item.(list.DefaultItem)
	if !ok {
		return
	}
	title = i.Title()
	desc = i.Description()

	if m.Width() <= 0 {
		return
	}

	textWidth := m.Width() - s.NormalTitle.GetPaddingLeft() - s.NormalTitle.GetPaddingRight()
	if textWidth < 1 {
		textWidth = 1
	}
	title = ansi.Truncate(title, textWidth, "...")
	if d.ShowDescription {
		maxDescLines := d.Height() - 1
		var lines []string
		for i, line := range strings.Split(desc, "\n") {
			if i >= maxDescLines {
				break
			}
			lines = append(lines, ansi.Truncate(line, textWidth, "..."))
		}
		desc = strings.Join(lines, "\n")
	}

	isSelected := index == m.Index()
	emptyFilter := m.FilterState() == list.Filtering && m.FilterValue() == ""
	isFiltered := m.FilterState() == list.Filtering || m.FilterState() == list.FilterApplied
	if isFiltered {
		matchedRunes = m.MatchesForItem(index)
	}

	if emptyFilter {
		title = s.DimmedTitle.Render(title)
		desc = s.DimmedDesc.Render(desc)
	} else if isSelected && m.FilterState() != list.Filtering {
		if isFiltered {
			unmatched := s.SelectedTitle.Inline(true)
			matched := unmatched.Inherit(s.FilterMatch)
			title = lipgloss.StyleRunes(title, matchedRunes, matched, unmatched)
		}
		title = d.fullRowStyle(m.Width(), s.SelectedTitle).Render(title)
		desc = s.SelectedDesc.Render(desc)
	} else {
		if isFiltered {
			unmatched := s.NormalTitle.Inline(true)
			matched := unmatched.Inherit(s.FilterMatch)
			title = lipgloss.StyleRunes(title, matchedRunes, matched, unmatched)
		}
		title = s.NormalTitle.Render(title)
		desc = s.NormalDesc.Render(desc)
	}

	if d.ShowDescription {
		fmt.Fprintf(w, "%s\n%s", title, desc) //nolint:errcheck
		return
	}
	fmt.Fprintf(w, "%s", title) //nolint:errcheck
}

func (d *fullRowDelegate) fullRowStyle(width int, style lipgloss.Style) lipgloss.Style {
	if width < 1 {
		width = 1
	}
	return style.Width(width)
}

func NewListModel() list.Model {
	delegate := newListDelegate()
	listModel := list.New([]list.Item{}, delegate, 0, 0)
	listModel.SetShowHelp(false)
	listModel.SetShowStatusBar(false)
	listModel.SetShowPagination(false)
	listModel.SetShowTitle(false)
	listModel.SetShowFilter(false)
	listModel.SetFilteringEnabled(false)
	listModel.DisableQuitKeybindings()
	return listModel
}
