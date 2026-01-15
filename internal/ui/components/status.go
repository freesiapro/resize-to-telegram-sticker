package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/resize-to-telegram-sticker/internal/ui/core"
)

func RenderStatus(styles core.Styles, contentWidth int, statusLine string, helpLine string) string {
	statusText := ansi.Truncate(statusLine, contentWidth, "...")
	statusView := styles.StatusBar.Width(contentWidth).Render(statusText)
	dividerView := styles.StatusDivider.Width(contentWidth).Render(core.HorizontalRule(contentWidth))
	helpText := ansi.Truncate(helpLine, contentWidth, "...")
	helpView := styles.HelpBar.Width(contentWidth).Render(helpText)
	return lipgloss.JoinVertical(lipgloss.Left, statusView, dividerView, helpView)
}
