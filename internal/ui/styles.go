package ui

import "github.com/charmbracelet/lipgloss"

type styles struct {
	headerFocused lipgloss.Style
	headerBlurred lipgloss.Style
	divider       lipgloss.Style
	statusBar     lipgloss.Style
	modal         lipgloss.Style
	modalTitle    lipgloss.Style
}

func newStyles() styles {
	return styles{
		headerFocused: lipgloss.NewStyle().Foreground(lipgloss.Color("69")).Bold(true).Padding(0, 1),
		headerBlurred: lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Padding(0, 1),
		divider:       lipgloss.NewStyle().Foreground(lipgloss.Color("238")),
		statusBar:     lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		modal: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2).
			BorderForeground(lipgloss.Color("69")),
		modalTitle: lipgloss.NewStyle().Bold(true),
	}
}
