package screens

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/resize-to-telegram-sticker/internal/ui/components"
	"github.com/resize-to-telegram-sticker/internal/ui/core"
)

type ConfigAction int

const (
	ConfigActionNone ConfigAction = iota
	ConfigActionBack
	ConfigActionStart
)

type ConfigUpdateResult struct {
	Cmd    tea.Cmd
	Action ConfigAction
}

type ConfigScreen struct {
	OutputDir   string
	OutputInput textinput.Model
}

func NewConfigScreen() ConfigScreen {
	output := textinput.New()
	output.SetValue("./output")
	output.Placeholder = "./output"

	return ConfigScreen{
		OutputDir:   "./output",
		OutputInput: output,
	}
}

func (c ConfigScreen) Update(msg tea.Msg) (ConfigScreen, ConfigUpdateResult) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			c.OutputInput.Blur()
			return c, ConfigUpdateResult{Action: ConfigActionBack}
		case "enter":
			value := strings.TrimSpace(c.OutputInput.Value())
			if value == "" {
				value = "./output"
				c.OutputInput.SetValue(value)
			}
			c.OutputDir = value
			c.OutputInput.Blur()
			return c, ConfigUpdateResult{Action: ConfigActionStart}
		}
	}

	var cmd tea.Cmd
	c.OutputInput, cmd = c.OutputInput.Update(msg)
	return c, ConfigUpdateResult{Cmd: cmd}
}

func (c *ConfigScreen) Focus() tea.Cmd {
	return c.OutputInput.Focus()
}

func (c ConfigScreen) View(width, height int, styles core.Styles) string {
	lines := []string{
		styles.ModalTitle.Render("Configuration"),
		"",
		"Output Directory:",
		c.OutputInput.View(),
		"",
		"Mode: Video Sticker",
		"",
		"Enter: Start  Esc: Back",
	}
	return components.RenderModal(styles, width, height, lines)
}
