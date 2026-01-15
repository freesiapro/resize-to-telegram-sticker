package screens

import (
	"github.com/resize-to-telegram-sticker/internal/ui/components"
	"github.com/resize-to-telegram-sticker/internal/ui/core"
)

type ProcessingScreen struct{}

func (p ProcessingScreen) View(width, height int, styles core.Styles) string {
	return components.RenderModal(styles, width, height, []string{"Processing..."})
}
