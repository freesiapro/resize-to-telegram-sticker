package ui

import "github.com/charmbracelet/lipgloss"

type styles struct {
	paneFocused lipgloss.Style
	paneBlurred lipgloss.Style
	header      lipgloss.Style
	statusBar   lipgloss.Style
	modal       lipgloss.Style
	modalTitle  lipgloss.Style
}

func newStyles() styles {
	return styles{
		paneFocused: lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("69")),
		paneBlurred: lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("238")),
		header:      lipgloss.NewStyle().Bold(true),
		statusBar:   lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		modal: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2).
			BorderForeground(lipgloss.Color("69")),
		modalTitle: lipgloss.NewStyle().Bold(true),
	}
}
