package screens

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/resize-to-telegram-sticker/internal/app"
	"github.com/resize-to-telegram-sticker/internal/domain"
	"github.com/resize-to-telegram-sticker/internal/infra"
	"github.com/resize-to-telegram-sticker/internal/ui/components"
	"github.com/resize-to-telegram-sticker/internal/ui/core"
)

type FocusArea int

const (
	FocusLeft FocusArea = iota
	FocusRight
)

const searchPrefix = "Search: "

type BrowseEventType int

const (
	BrowseEventNone BrowseEventType = iota
	BrowseEventOpenDir
	BrowseEventStartConfirm
)

type BrowseEvent struct {
	Type BrowseEventType
	Path string
}

type BrowseUpdateResult struct {
	Cmd   tea.Cmd
	Event BrowseEvent
}

type BrowseScreen struct {
	Focus FocusArea

	Width  int
	Height int

	Cwd         string
	FilterInput textinput.Model
	Entries     []core.EntryItem

	SelectedSet  map[string]struct{}
	SelectedList []app.SelectionItem

	LeftList  list.Model
	RightList list.Model

	Status string
}

func NewBrowseScreen(cwd string) BrowseScreen {
	left := core.NewListModel()
	right := core.NewListModel()

	filter := textinput.New()
	filter.Prompt = ""
	filter.Focus()
	filter.TextStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("24"))
	filter.PlaceholderStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Background(lipgloss.Color("24"))
	filter.Cursor.TextStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("24"))

	return BrowseScreen{
		Focus:        FocusLeft,
		Cwd:          cwd,
		FilterInput:  filter,
		Entries:      make([]core.EntryItem, 0),
		SelectedSet:  make(map[string]struct{}),
		SelectedList: make([]app.SelectionItem, 0),
		LeftList:     left,
		RightList:    right,
	}
}

func (b BrowseScreen) Update(msg tea.Msg) (BrowseScreen, BrowseUpdateResult) {
	if _, ok := msg.(tea.KeyMsg); !ok {
		var cmd tea.Cmd
		b.FilterInput, cmd = b.FilterInput.Update(msg)
		return b, BrowseUpdateResult{Cmd: cmd}
	}

	key := msg.(tea.KeyMsg)
	switch key.String() {
	case "tab":
		if b.Focus == FocusLeft {
			b.Focus = FocusRight
			b.FilterInput.Blur()
		} else {
			b.Focus = FocusLeft
			return b, BrowseUpdateResult{Cmd: b.FilterInput.Focus()}
		}
		return b, BrowseUpdateResult{}
	case "enter":
		if b.Focus == FocusLeft {
			item, ok := b.LeftList.SelectedItem().(leftListItem)
			if !ok || !item.entry.IsDir {
				return b, BrowseUpdateResult{}
			}
			b.FilterInput.Reset()
			return b, BrowseUpdateResult{
				Event: BrowseEvent{Type: BrowseEventOpenDir, Path: item.entry.Path},
			}
		}
		return b.startConfirm()
	case " ":
		if b.Focus == FocusLeft {
			item, ok := b.LeftList.SelectedItem().(leftListItem)
			if ok {
				b.toggleSelection(item.entry)
			}
		}
		return b, BrowseUpdateResult{}
	case "backspace":
		if b.Focus == FocusRight {
			b.removeSelectedRight()
			return b, BrowseUpdateResult{}
		}
	}

	cmds := make([]tea.Cmd, 0, 2)
	if b.Focus == FocusLeft {
		before := b.FilterInput.Value()
		var cmd tea.Cmd
		b.FilterInput, cmd = b.FilterInput.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		if b.FilterInput.Value() != before {
			b.refreshLeftList()
		}
	}

	shouldUpdateList := true
	if b.Focus == FocusLeft {
		if key.Type == tea.KeyRunes || key.String() == "backspace" {
			shouldUpdateList = false
		}
	}
	if shouldUpdateList {
		var cmd tea.Cmd
		if b.Focus == FocusLeft {
			b.LeftList, cmd = b.LeftList.Update(msg)
		} else {
			b.RightList, cmd = b.RightList.Update(msg)
		}
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return b, BrowseUpdateResult{Cmd: batchCmds(cmds)}
}

func (b *BrowseScreen) ApplyDirEntries(path string, entries []infra.DirEntry) {
	b.Cwd = path
	b.Entries = buildEntries(path, entries, b.SelectedSet)
	b.refreshLeftList()
}

func (b *BrowseScreen) SetStatus(err error) {
	if err == nil {
		b.Status = ""
		return
	}
	b.Status = fmt.Sprintf("Error: %v", err)
}

func (b *BrowseScreen) Resize(width, height int) {
	b.Width = width
	b.Height = height
	if width == 0 || height == 0 {
		return
	}
	contentWidth, contentHeight := core.ContentSize(width, height)
	layout := core.CalcPaneLayout(contentWidth, contentHeight)
	b.LeftList.SetSize(layout.LeftInnerWidth, layout.ListHeight)
	b.RightList.SetSize(layout.RightInnerWidth, layout.ListHeight)
	headerWidth := layout.LeftInnerWidth
	filterWidth := headerWidth - len(searchPrefix)
	if filterWidth < 1 {
		filterWidth = 1
	}
	b.FilterInput.Width = filterWidth
}

func (b BrowseScreen) View(width, height int, styles core.Styles) string {
	contentWidth, contentHeight := core.ContentSize(width, height)
	layout := core.CalcPaneLayout(contentWidth, contentHeight)
	leftHeaderWidth := layout.LeftInnerWidth
	rightHeaderWidth := layout.RightInnerWidth

	filterView := b.FilterInput.View()
	leftHeaderText := leftHeaderLine(filterView, leftHeaderWidth)
	rightHeaderText := rightHeaderLine(len(b.SelectedList))
	rightHeaderText = ansi.Truncate(rightHeaderText, rightHeaderWidth, "...")

	leftHeaderStyle := styles.HeaderBlurred
	rightHeaderStyle := styles.HeaderBlurred
	if b.Focus == FocusLeft {
		leftHeaderStyle = styles.HeaderFocused
	}
	if b.Focus == FocusRight {
		rightHeaderStyle = styles.HeaderFocused
	}

	leftHeader := leftHeaderStyle.Width(leftHeaderWidth).Render(leftHeaderText)
	rightHeader := rightHeaderStyle.Width(rightHeaderWidth).Render(rightHeaderText)

	top := components.RenderPane(styles, layout, components.PaneProps{
		LeftHeader:  leftHeader,
		RightHeader: rightHeader,
		LeftBody:    b.LeftList.View(),
		RightBody:   b.RightList.View(),
		FocusLeft:   b.Focus == FocusLeft,
		FocusRight:  b.Focus == FocusRight,
	})

	status := components.RenderStatus(styles, contentWidth, b.statusLine(), b.helpLine(styles))
	content := lipgloss.JoinVertical(lipgloss.Left, top, status)
	return styles.Outer.Width(width).Height(height).Render(content)
}

func (b BrowseScreen) SelectedItems() []app.SelectionItem {
	return b.SelectedList
}

func (b BrowseScreen) startConfirm() (BrowseScreen, BrowseUpdateResult) {
	if len(b.SelectedList) == 0 {
		b.Status = "No selection"
		return b, BrowseUpdateResult{}
	}
	b.Status = ""
	return b, BrowseUpdateResult{Event: BrowseEvent{Type: BrowseEventStartConfirm}}
}

func (b *BrowseScreen) toggleSelection(entry core.EntryItem) {
	if entry.IsParent {
		return
	}
	if _, ok := b.SelectedSet[entry.Path]; ok {
		delete(b.SelectedSet, entry.Path)
		b.SelectedList = removeSelection(b.SelectedList, entry.Path)
	} else {
		b.SelectedSet[entry.Path] = struct{}{}
		b.SelectedList = append(b.SelectedList, app.SelectionItem{Path: entry.Path, IsDir: entry.IsDir})
	}
	b.refreshLeftList()
	b.refreshRightList()
}

func (b *BrowseScreen) removeSelectedRight() {
	item, ok := b.RightList.SelectedItem().(rightListItem)
	if !ok {
		return
	}
	delete(b.SelectedSet, item.selection.Path)
	b.SelectedList = removeSelection(b.SelectedList, item.selection.Path)
	b.refreshLeftList()
	b.refreshRightList()
}

func (b *BrowseScreen) refreshLeftList() {
	filtered := core.FilterEntries(b.Entries, b.FilterInput.Value())
	items := make([]list.Item, 0, len(filtered))
	for _, e := range filtered {
		e.Selected = false
		if _, ok := b.SelectedSet[e.Path]; ok {
			e.Selected = true
		}
		items = append(items, leftListItem{entry: e})
	}
	b.LeftList.SetItems(items)
}

func (b *BrowseScreen) refreshRightList() {
	items := make([]list.Item, 0, len(b.SelectedList))
	for _, s := range b.SelectedList {
		items = append(items, rightListItem{selection: s, display: formatSelection(b.Cwd, s)})
	}
	b.RightList.SetItems(items)
}

func (b BrowseScreen) statusLine() string {
	focus := "Left"
	if b.Focus == FocusRight {
		focus = "Right"
	}
	parts := []string{
		fmt.Sprintf("Focus: %s", focus),
		fmt.Sprintf("Selected: %d", len(b.SelectedList)),
	}
	if b.Status != "" {
		parts = append([]string{fmt.Sprintf("Status: %s", b.Status)}, parts...)
	}
	return strings.Join(parts, " | ")
}

func (b BrowseScreen) helpLine(styles core.Styles) string {
	hints := []keyHint{
		{key: "Tab", action: "Switch"},
		{key: "Enter", action: "Open/Next"},
		{key: "Space", action: "Toggle"},
		{key: "Backspace", action: "Remove"},
		{key: "q", action: "Quit"},
	}
	parts := make([]string, 0, len(hints))
	for _, hint := range hints {
		key := styles.HintKey.Render(hint.key)
		action := styles.HintAction.Render(hint.action)
		parts = append(parts, fmt.Sprintf("%s %s", key, action))
	}
	return strings.Join(parts, "  ")
}

func leftHeaderLine(filterView string, width int) string {
	line := fmt.Sprintf("%s%s", searchPrefix, filterView)
	if width <= 0 {
		return ""
	}
	return ansi.Truncate(line, width, "")
}

func rightHeaderLine(count int) string {
	return fmt.Sprintf("Selected: %d", count)
}

func buildEntries(cwd string, entries []infra.DirEntry, selected map[string]struct{}) []core.EntryItem {
	items := make([]core.EntryItem, 0)
	parent := filepath.Dir(cwd)
	if parent != cwd {
		items = append(items, core.EntryItem{Path: parent, Name: "..", IsDir: true, IsParent: true})
	}

	dirs := make([]core.EntryItem, 0)
	files := make([]core.EntryItem, 0)
	for _, e := range entries {
		if e.IsDir {
			dirs = append(dirs, core.EntryItem{Path: e.Path, Name: e.Name, IsDir: true})
			continue
		}
		if _, err := domain.DetectInputKind(e.Path); err != nil {
			continue
		}
		files = append(files, core.EntryItem{Path: e.Path, Name: e.Name, IsDir: false})
	}

	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name < dirs[j].Name })
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })

	items = append(items, dirs...)
	items = append(items, files...)

	for i := range items {
		if _, ok := selected[items[i].Path]; ok {
			items[i].Selected = true
		}
	}

	return items
}

func removeSelection(list []app.SelectionItem, path string) []app.SelectionItem {
	out := make([]app.SelectionItem, 0, len(list))
	for _, s := range list {
		if s.Path == path {
			continue
		}
		out = append(out, s)
	}
	return out
}

func formatSelection(cwd string, item app.SelectionItem) string {
	name := item.Path
	rel, err := filepath.Rel(cwd, item.Path)
	if err == nil && !strings.HasPrefix(rel, "..") {
		name = filepath.ToSlash(filepath.Join(".", rel))
	}
	if item.IsDir {
		return name + "/"
	}
	return name
}

func batchCmds(cmds []tea.Cmd) tea.Cmd {
	if len(cmds) == 0 {
		return nil
	}
	if len(cmds) == 1 {
		return cmds[0]
	}
	return tea.Batch(cmds...)
}

type leftListItem struct {
	entry core.EntryItem
}

func (i leftListItem) Title() string {
	mark := "[ ]"
	if i.entry.Selected {
		mark = "[x]"
	}
	name := i.entry.Name
	if i.entry.IsParent {
		name = "../"
	} else if i.entry.IsDir {
		name += "/"
	}
	return fmt.Sprintf("%s %s", mark, name)
}

func (i leftListItem) Description() string { return "" }
func (i leftListItem) FilterValue() string { return i.entry.Name }

type rightListItem struct {
	selection app.SelectionItem
	display   string
}

func (i rightListItem) Title() string       { return i.display }
func (i rightListItem) Description() string { return "" }
func (i rightListItem) FilterValue() string { return i.display }

type keyHint struct {
	key    string
	action string
}
