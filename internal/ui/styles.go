package ui

import "github.com/charmbracelet/lipgloss"

type styles struct {
	paneFocused   lipgloss.Style
	paneBlurred   lipgloss.Style
	headerFocused lipgloss.Style
	headerBlurred lipgloss.Style
	divider       lipgloss.Style
	statusBar     lipgloss.Style
	outer         lipgloss.Style
	modal         lipgloss.Style
	modalTitle    lipgloss.Style
}

func newStyles() styles {
	return styles{
		paneFocused: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("69")),
		paneBlurred: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("238")),
		headerFocused: lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("24")).
			Bold(true).
			Padding(0, 1),
		headerBlurred: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Background(lipgloss.Color("235")).
			Padding(0, 1),
		divider:   lipgloss.NewStyle().Foreground(lipgloss.Color("238")),
		statusBar: lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		outer:     lipgloss.NewStyle().Padding(outerPadY, outerPadX),
		modal: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2).
			BorderForeground(lipgloss.Color("69")),
		modalTitle: lipgloss.NewStyle().Bold(true),
	}
}
