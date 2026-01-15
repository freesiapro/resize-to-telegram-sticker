package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/resize-to-telegram-sticker/internal/ui/core"
)

func RenderModal(styles core.Styles, width, height int, lines []string) string {
	contentWidth, contentHeight := core.ContentSize(width, height)
	box := styles.Modal.Render(strings.Join(lines, "\n"))
	placed := lipgloss.Place(contentWidth, contentHeight, lipgloss.Center, lipgloss.Center, box)
	return styles.Outer.Width(width).Height(height).Render(placed)
}
