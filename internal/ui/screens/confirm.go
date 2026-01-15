package screens

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/resize-to-telegram-sticker/internal/app"
	"github.com/resize-to-telegram-sticker/internal/ui/components"
	"github.com/resize-to-telegram-sticker/internal/ui/core"
)

type ConfirmAction int

const (
	ConfirmActionNone ConfirmAction = iota
	ConfirmActionBack
	ConfirmActionContinue
)

type ConfirmUpdateResult struct {
	Cmd    tea.Cmd
	Action ConfirmAction
}

type ConfirmScreen struct {
	Loading bool
	Result  app.ExpandResult
	Err     error
}

func (c ConfirmScreen) Update(msg tea.Msg) (ConfirmScreen, ConfirmUpdateResult) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return c, ConfirmUpdateResult{}
	}
	switch key.String() {
	case "esc":
		return c, ConfirmUpdateResult{Action: ConfirmActionBack}
	case "enter":
		if c.Loading || c.Err != nil {
			return c, ConfirmUpdateResult{}
		}
		return c, ConfirmUpdateResult{Action: ConfirmActionContinue}
	}
	return c, ConfirmUpdateResult{}
}

func (c ConfirmScreen) View(width, height int, styles core.Styles) string {
	if c.Loading {
		return components.RenderModal(styles, width, height, []string{"Scanning directories..."})
	}
	if c.Err != nil {
		lines := []string{
			fmt.Sprintf("Error: %v", c.Err),
			"",
			"Esc: Back",
		}
		return components.RenderModal(styles, width, height, lines)
	}

	lines := []string{
		styles.ModalTitle.Render("Confirm Selection"),
		"",
		fmt.Sprintf("Directories: %d", c.Result.DirCount),
		fmt.Sprintf("Files: %d", c.Result.FileCount),
		fmt.Sprintf("Total files: %d", c.Result.TotalFiles),
		"",
		"Output dirs:",
	}
	for _, d := range c.Result.OutputDirs {
		lines = append(lines, "- "+d)
	}
	lines = append(lines, "", "Enter: Continue  Esc: Back")
	return components.RenderModal(styles, width, height, lines)
}

func (c *ConfirmScreen) StartLoading() {
	c.Loading = true
	c.Err = nil
}

func (c *ConfirmScreen) SetResult(result app.ExpandResult, err error) {
	c.Loading = false
	c.Err = err
	if err == nil {
		c.Result = result
	}
}
