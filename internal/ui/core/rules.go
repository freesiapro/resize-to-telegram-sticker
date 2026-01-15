package core

import "strings"

func VerticalRule(height int) string {
	if height < 1 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("|\n", height), "\n")
}

func HorizontalRule(width int) string {
	if width < 1 {
		return ""
	}
	return strings.Repeat("-", width)
}
