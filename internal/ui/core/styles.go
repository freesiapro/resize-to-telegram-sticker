package core

import "github.com/charmbracelet/lipgloss"

type Styles struct {
	PaneFocused   lipgloss.Style
	PaneBlurred   lipgloss.Style
	HeaderFocused lipgloss.Style
	HeaderBlurred lipgloss.Style
	Divider       lipgloss.Style
	StatusBar     lipgloss.Style
	HelpBar       lipgloss.Style
	StatusDivider lipgloss.Style
	HintKey       lipgloss.Style
	HintAction    lipgloss.Style
	Outer         lipgloss.Style
	Modal         lipgloss.Style
	ModalTitle    lipgloss.Style
}

func NewStyles() Styles {
	return Styles{
		PaneFocused: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("69")),
		PaneBlurred: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("238")),
		HeaderFocused: lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("24")).
			Bold(true).
			Padding(0, headerPadX),
		HeaderBlurred: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Background(lipgloss.Color("235")).
			Padding(0, headerPadX),
		Divider:       lipgloss.NewStyle().Foreground(lipgloss.Color("238")),
		StatusBar:     lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		HelpBar:       lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		StatusDivider: lipgloss.NewStyle().Foreground(lipgloss.Color("238")),
		HintKey:       lipgloss.NewStyle().Foreground(lipgloss.Color("69")).Bold(true),
		HintAction:    lipgloss.NewStyle().Foreground(lipgloss.Color("246")),
		Outer:         lipgloss.NewStyle().Padding(outerPadY, outerPadX),
		Modal: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2).
			BorderForeground(lipgloss.Color("69")),
		ModalTitle: lipgloss.NewStyle().Bold(true),
	}
}
