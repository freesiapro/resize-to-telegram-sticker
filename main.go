package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/resize-to-telegram-sticker/internal/ui"
)

func main() {
	p := tea.NewProgram(ui.NewModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("run failed: %v\n", err)
		os.Exit(1)
	}
}
