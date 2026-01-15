package screens

import (
	"fmt"

	"github.com/resize-to-telegram-sticker/internal/app"
	"github.com/resize-to-telegram-sticker/internal/ui/components"
	"github.com/resize-to-telegram-sticker/internal/ui/core"
)

type SummaryScreen struct {
	Results []app.Result
}

func (s SummaryScreen) View(width, height int, styles core.Styles) string {
	success := 0
	failed := 0
	for _, r := range s.Results {
		if r.Err != nil || len(r.Issues) > 0 {
			failed++
			continue
		}
		success++
	}
	lines := []string{
		styles.ModalTitle.Render("Summary"),
		"",
		fmt.Sprintf("Success: %d", success),
		fmt.Sprintf("Failed: %d", failed),
		"",
		"Press q to quit",
	}
	return components.RenderModal(styles, width, height, lines)
}

func (s *SummaryScreen) SetResults(results []app.Result) {
	s.Results = results
}
