package components

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/resize-to-telegram-sticker/internal/ui/core"
)

type PaneProps struct {
	LeftHeader  string
	RightHeader string
	LeftBody    string
	RightBody   string
	FocusLeft   bool
	FocusRight  bool
}

func RenderPane(styles core.Styles, layout core.PaneLayout, props PaneProps) string {
	leftContent := lipgloss.JoinVertical(lipgloss.Left, props.LeftHeader, props.LeftBody)
	rightContent := lipgloss.JoinVertical(lipgloss.Left, props.RightHeader, props.RightBody)
	contentLimiter := lipgloss.NewStyle().MaxHeight(layout.InnerHeight)
	leftContent = contentLimiter.Render(leftContent)
	rightContent = contentLimiter.Render(rightContent)

	leftPaneStyle := lipgloss.NewStyle()
	rightPaneStyle := lipgloss.NewStyle()
	if layout.BorderSize > 0 {
		leftPaneStyle = styles.PaneBlurred
		rightPaneStyle = styles.PaneBlurred
		if props.FocusLeft {
			leftPaneStyle = styles.PaneFocused
		}
		if props.FocusRight {
			rightPaneStyle = styles.PaneFocused
		}
	}

	leftView := leftPaneStyle.Width(layout.LeftInnerWidth).Height(layout.InnerHeight).Render(leftContent)
	rightView := rightPaneStyle.Width(layout.RightInnerWidth).Height(layout.InnerHeight).Render(rightContent)

	var divider string
	if layout.DividerWidth > 0 {
		divider = styles.Divider.Render(core.VerticalRule(layout.ContentHeight))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, leftView, divider, rightView)
}
