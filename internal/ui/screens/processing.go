package screens

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/x/ansi"

	"github.com/resize-to-telegram-sticker/internal/app"
	"github.com/resize-to-telegram-sticker/internal/ui/core"
)

type ProcessingStatus int

const (
	ProcessingPending ProcessingStatus = iota
	ProcessingRunning
	ProcessingDone
	ProcessingFailed
)

type ProcessingItem struct {
	ID     int
	Path   string
	Status ProcessingStatus
	Err    string
}

type ProcessingScreen struct {
	Items     []ProcessingItem
	DoneCount int
	progress  progress.Model
}

func NewProcessingScreen() ProcessingScreen {
	return ProcessingScreen{
		Items:    make([]ProcessingItem, 0),
		progress: progress.New(progress.WithSolidFill("69")),
	}
}

func (p *ProcessingScreen) Reset() {
	p.Items = make([]ProcessingItem, 0)
	p.DoneCount = 0
}

func (p *ProcessingScreen) SetJobs(jobs []app.Job) {
	items := make([]ProcessingItem, 0, len(jobs))
	for i, job := range jobs {
		items = append(items, ProcessingItem{
			ID:     i,
			Path:   job.InputPath,
			Status: ProcessingPending,
		})
	}
	p.Items = items
	p.DoneCount = 0
}

func (p *ProcessingScreen) MarkProcessing(index int) {
	if index < 0 || index >= len(p.Items) {
		return
	}
	p.Items[index].Status = ProcessingRunning
}

func (p *ProcessingScreen) ApplyResult(index int, result app.Result) {
	if index < 0 || index >= len(p.Items) {
		return
	}
	previous := p.Items[index].Status
	status, message := resultStatus(result)
	p.Items[index].Status = status
	p.Items[index].Err = message
	if previous != ProcessingDone && previous != ProcessingFailed {
		if status == ProcessingDone || status == ProcessingFailed {
			p.DoneCount++
		}
	}
}

func (p ProcessingScreen) NextPendingIndex() int {
	for i, item := range p.Items {
		if item.Status == ProcessingPending {
			return i
		}
	}
	return -1
}

func (p ProcessingScreen) View(width, height int, styles core.Styles) string {
	contentWidth, contentHeight := core.ContentSize(width, height)
	if contentWidth < 1 {
		contentWidth = 1
	}
	lines := make([]string, 0)
	lines = append(lines, styles.ModalTitle.Render("Processing"))
	lines = append(lines, truncateLine(fmt.Sprintf("Done: %d / %d", p.DoneCount, len(p.Items)), contentWidth))
	lines = append(lines, truncateLine(fmt.Sprintf("Current: %s", p.currentLabel()), contentWidth))
	lines = append(lines, p.progressLine(contentWidth))
	lines = append(lines, "")

	listHeight := contentHeight - len(lines)
	if listHeight < 0 {
		listHeight = 0
	}
	lines = append(lines, p.listLines(listHeight, contentWidth)...)
	content := strings.Join(lines, "\n")
	return styles.Outer.Width(width).Height(height).Render(content)
}

func (p ProcessingScreen) progressLine(width int) string {
	model := p.progress
	model.Width = width
	return model.ViewAs(p.progressPercent())
}

func (p ProcessingScreen) progressPercent() float64 {
	if len(p.Items) == 0 {
		return 0
	}
	return float64(p.DoneCount) / float64(len(p.Items))
}

func (p ProcessingScreen) currentLabel() string {
	processing := make([]ProcessingItem, 0)
	for _, item := range p.Items {
		if item.Status == ProcessingRunning {
			processing = append(processing, item)
		}
	}
	if len(processing) == 0 {
		return "-"
	}
	if len(processing) == 1 {
		return processing[0].Path
	}
	return fmt.Sprintf("%s (+%d)", processing[0].Path, len(processing)-1)
}

func (p ProcessingScreen) listLines(height int, width int) []string {
	if height <= 0 {
		return nil
	}
	items := p.sortedItems()
	lines := make([]string, 0, height)
	for _, item := range items {
		if len(lines) >= height {
			break
		}
		line := formatProcessingItem(item)
		lines = append(lines, ansi.Truncate(line, width, "..."))
	}
	return lines
}

func (p ProcessingScreen) sortedItems() []ProcessingItem {
	items := make([]ProcessingItem, len(p.Items))
	copy(items, p.Items)
	sort.SliceStable(items, func(i, j int) bool {
		leftRank := statusRank(items[i].Status)
		rightRank := statusRank(items[j].Status)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		return items[i].ID < items[j].ID
	})
	return items
}

func statusRank(status ProcessingStatus) int {
	switch status {
	case ProcessingRunning:
		return 0
	case ProcessingPending:
		return 1
	default:
		return 2
	}
}

func formatProcessingItem(item ProcessingItem) string {
	label := "[TODO]"
	switch item.Status {
	case ProcessingRunning:
		label = "[RUN]"
	case ProcessingDone:
		label = "[DONE]"
	case ProcessingFailed:
		label = "[FAIL]"
	}
	line := fmt.Sprintf("%s %s", label, item.Path)
	if item.Err != "" {
		line = fmt.Sprintf("%s (%s)", line, sanitizeLine(item.Err))
	}
	return line
}

func sanitizeLine(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	return strings.ReplaceAll(trimmed, "\n", " ")
}

func truncateLine(value string, width int) string {
	if width <= 0 {
		return ""
	}
	return ansi.Truncate(value, width, "...")
}

func resultStatus(result app.Result) (ProcessingStatus, string) {
	if result.Err != nil {
		return ProcessingFailed, result.Err.Error()
	}
	if len(result.Issues) > 0 {
		return ProcessingFailed, result.Issues[0].Message
	}
	return ProcessingDone, ""
}
