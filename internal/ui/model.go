package ui

import (
	"context"
	"fmt"
	"os"
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
)

type viewState int

type focusArea int

const (
	stateBrowse viewState = iota
	stateConfirm
	stateConfig
	stateProcessing
	stateSummary
)

const (
	focusLeft focusArea = iota
	focusRight
)

const searchPrefix = "Search: "

type DirLister interface {
	ListDirEntries(root string) ([]infra.DirEntry, error)
}

type SelectionExpander interface {
	Expand(selections []app.SelectionItem, outputDir string) (app.ExpandResult, error)
}

type dirLister struct{}

func (dirLister) ListDirEntries(root string) ([]infra.DirEntry, error) {
	return infra.ListDirEntries(root)
}

type model struct {
	state viewState
	focus focusArea

	width  int
	height int

	cwd         string
	filterInput textinput.Model
	entries     []entryItem

	selectedSet  map[string]struct{}
	selectedList []app.SelectionItem

	leftList  list.Model
	rightList list.Model

	confirmLoading bool
	confirmResult  app.ExpandResult
	confirmErr     error

	outputDir   string
	outputInput textinput.Model

	results []app.Result
	status  string
	err     error

	lister   DirLister
	expander SelectionExpander
	pipeline app.Pipeline

	styles styles
}

func NewModel() model {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	return NewModelWithDeps(cwd, dirLister{}, app.SelectionExpander{})
}

func NewModelWithDeps(cwd string, lister DirLister, expander SelectionExpander) model {
	left := newListModel()
	right := newListModel()

	filter := textinput.New()
	filter.Prompt = ""
	filter.Focus()

	output := textinput.New()
	output.SetValue("./output")
	output.Placeholder = "./output"

	return model{
		state:        stateBrowse,
		focus:        focusLeft,
		cwd:          cwd,
		filterInput:  filter,
		selectedSet:  make(map[string]struct{}),
		selectedList: make([]app.SelectionItem, 0),
		leftList:     left,
		rightList:    right,
		outputDir:    "./output",
		outputInput:  output,
		lister:       lister,
		expander:     expander,
		pipeline: app.Pipeline{
			Probe:  infra.FFprobeRunner{},
			Encode: infra.FFmpegRunner{},
		},
		styles: newStyles(),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(loadDirCmd(m.cwd, m.lister), textinput.Blink)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var inputCmd tea.Cmd
	if _, ok := msg.(tea.KeyMsg); !ok {
		switch m.state {
		case stateBrowse:
			m.filterInput, inputCmd = m.filterInput.Update(msg)
		case stateConfig:
			m.outputInput, inputCmd = m.outputInput.Update(msg)
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeLists()
		return m, inputCmd
	case dirEntriesMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Error: %v", msg.err)
			return m, inputCmd
		}
		m.cwd = msg.path
		m.entries = buildEntries(msg.path, msg.entries, m.selectedSet)
		m.refreshLeftList()
		return m, inputCmd
	case confirmMsg:
		m.confirmLoading = false
		m.confirmErr = msg.err
		if msg.err == nil {
			m.confirmResult = msg.result
		}
		return m, inputCmd
	case errMsg:
		m.status = fmt.Sprintf("Error: %v", msg.err)
		return m, inputCmd
	case doneMsg:
		m.state = stateSummary
		m.results = msg.results
		return m, inputCmd
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
		switch m.state {
		case stateBrowse:
			updated, cmd := m.updateBrowse(msg)
			return updated, tea.Batch(inputCmd, cmd)
		case stateConfirm:
			updated, cmd := m.updateConfirm(msg)
			return updated, tea.Batch(inputCmd, cmd)
		case stateConfig:
			updated, cmd := m.updateConfig(msg)
			return updated, tea.Batch(inputCmd, cmd)
		case stateSummary:
			return m, inputCmd
		case stateProcessing:
			return m, inputCmd
		}
	}

	return m, inputCmd
}

func (m model) View() string {
	switch m.state {
	case stateConfirm:
		return m.viewConfirm()
	case stateConfig:
		return m.viewConfig()
	case stateProcessing:
		return m.viewProcessing()
	case stateSummary:
		return m.viewSummary()
	default:
		return m.viewBrowse()
	}
}

func (m model) updateBrowse(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		if m.focus == focusLeft {
			m.focus = focusRight
			m.filterInput.Blur()
		} else {
			m.focus = focusLeft
			return m, m.filterInput.Focus()
		}
		return m, nil
	case "enter":
		if m.focus == focusLeft {
			item, ok := m.leftList.SelectedItem().(leftListItem)
			if !ok || !item.entry.isDir {
				return m, nil
			}
			m.filterInput.Reset()
			return m, loadDirCmd(item.entry.path, m.lister)
		}
		return m.startConfirm()
	case " ":
		if m.focus == focusLeft {
			item, ok := m.leftList.SelectedItem().(leftListItem)
			if ok {
				m.toggleSelection(item.entry)
			}
		}
		return m, nil
	case "backspace":
		if m.focus == focusRight {
			m.removeSelectedRight()
			return m, nil
		}
	}

	var cmds []tea.Cmd
	if m.focus == focusLeft {
		before := m.filterInput.Value()
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		if m.filterInput.Value() != before {
			m.refreshLeftList()
		}
	}

	shouldUpdateList := true
	if m.focus == focusLeft {
		if msg.Type == tea.KeyRunes || msg.String() == "backspace" {
			shouldUpdateList = false
		}
	}
	if shouldUpdateList {
		updated, cmd := m.updateLists(msg)
		m = updated.(model)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if len(cmds) == 0 {
		return m, nil
	}
	return m, tea.Batch(cmds...)
}

func (m model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = stateBrowse
		return m, nil
	case "enter":
		if m.confirmLoading || m.confirmErr != nil {
			return m, nil
		}
		m.state = stateConfig
		m.outputInput.Focus()
		return m, nil
	}
	return m, nil
}

func (m model) updateConfig(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = stateBrowse
		m.outputInput.Blur()
		return m, nil
	case "enter":
		m.outputDir = strings.TrimSpace(m.outputInput.Value())
		if m.outputDir == "" {
			m.outputDir = "./output"
			m.outputInput.SetValue(m.outputDir)
		}
		m.state = stateProcessing
		m.outputInput.Blur()
		return m, runPipelineCmd(m.pipeline, m.expander, m.selectedList, m.outputDir)
	}

	var cmd tea.Cmd
	m.outputInput, cmd = m.outputInput.Update(msg)
	return m, cmd
}

func (m model) startConfirm() (tea.Model, tea.Cmd) {
	if len(m.selectedList) == 0 {
		m.err = fmt.Errorf("no selection")
		m.status = "No selection"
		return m, nil
	}
	m.err = nil
	m.status = ""
	m.state = stateConfirm
	m.confirmLoading = true
	m.confirmErr = nil
	return m, expandCmd(m.expander, m.selectedList, m.outputDir)
}

func (m *model) toggleSelection(entry entryItem) {
	if entry.isParent {
		return
	}
	if _, ok := m.selectedSet[entry.path]; ok {
		delete(m.selectedSet, entry.path)
		m.selectedList = removeSelection(m.selectedList, entry.path)
	} else {
		m.selectedSet[entry.path] = struct{}{}
		m.selectedList = append(m.selectedList, app.SelectionItem{Path: entry.path, IsDir: entry.isDir})
	}
	m.refreshLeftList()
	m.refreshRightList()
}

func (m *model) removeSelectedRight() {
	item, ok := m.rightList.SelectedItem().(rightListItem)
	if !ok {
		return
	}
	delete(m.selectedSet, item.selection.Path)
	m.selectedList = removeSelection(m.selectedList, item.selection.Path)
	m.refreshLeftList()
	m.refreshRightList()
}

func (m *model) refreshLeftList() {
	filtered := filterEntries(m.entries, m.filterInput.Value())
	items := make([]list.Item, 0, len(filtered))
	for _, e := range filtered {
		e.selected = false
		if _, ok := m.selectedSet[e.path]; ok {
			e.selected = true
		}
		items = append(items, leftListItem{entry: e})
	}
	m.leftList.SetItems(items)
}

func (m *model) refreshRightList() {
	items := make([]list.Item, 0, len(m.selectedList))
	for _, s := range m.selectedList {
		items = append(items, rightListItem{selection: s, display: formatSelection(m.cwd, s)})
	}
	m.rightList.SetItems(items)
}

func (m model) updateLists(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.focus == focusLeft {
		m.leftList, cmd = m.leftList.Update(msg)
		return m, cmd
	}
	m.rightList, cmd = m.rightList.Update(msg)
	return m, cmd
}

func (m *model) resizeLists() {
	if m.width == 0 || m.height == 0 {
		return
	}
	contentWidth, contentHeight := contentSize(m.width, m.height)
	layout := calcPaneLayout(contentWidth, contentHeight)
	m.leftList.SetSize(layout.leftInnerWidth, layout.listHeight)
	m.rightList.SetSize(layout.rightInnerWidth, layout.listHeight)
	headerWidth := headerContentWidth(layout.leftInnerWidth)
	filterWidth := headerWidth - len(searchPrefix) - 1
	if filterWidth < 1 {
		filterWidth = 1
	}
	m.filterInput.Width = filterWidth
}

func (m model) viewBrowse() string {
	contentWidth, contentHeight := contentSize(m.width, m.height)
	layout := calcPaneLayout(contentWidth, contentHeight)
	leftHeaderWidth := headerContentWidth(layout.leftInnerWidth)
	rightHeaderWidth := headerContentWidth(layout.rightInnerWidth)

	filterView := m.filterInput.View()
	leftHeaderText := leftHeaderLine(filterView, leftHeaderWidth)
	rightHeaderText := rightHeaderLine(len(m.selectedList))
	rightHeaderText = ansi.Truncate(rightHeaderText, rightHeaderWidth, "...")

	leftHeaderStyle := m.styles.headerBlurred
	rightHeaderStyle := m.styles.headerBlurred
	if m.focus == focusLeft {
		leftHeaderStyle = m.styles.headerFocused
	}
	if m.focus == focusRight {
		rightHeaderStyle = m.styles.headerFocused
	}

	leftHeader := leftHeaderStyle.Width(leftHeaderWidth).Render(leftHeaderText)
	rightHeader := rightHeaderStyle.Width(rightHeaderWidth).Render(rightHeaderText)

	leftContent := lipgloss.JoinVertical(lipgloss.Left, leftHeader, m.leftList.View())
	rightContent := lipgloss.JoinVertical(lipgloss.Left, rightHeader, m.rightList.View())
	contentLimiter := lipgloss.NewStyle().MaxHeight(layout.innerHeight)
	leftContent = contentLimiter.Render(leftContent)
	rightContent = contentLimiter.Render(rightContent)

	leftPaneStyle := lipgloss.NewStyle()
	rightPaneStyle := lipgloss.NewStyle()
	if layout.borderSize > 0 {
		leftPaneStyle = m.styles.paneBlurred
		rightPaneStyle = m.styles.paneBlurred
		if m.focus == focusLeft {
			leftPaneStyle = m.styles.paneFocused
		}
		if m.focus == focusRight {
			rightPaneStyle = m.styles.paneFocused
		}
	}

	leftView := leftPaneStyle.Width(layout.leftInnerWidth).Height(layout.innerHeight).Render(leftContent)
	rightView := rightPaneStyle.Width(layout.rightInnerWidth).Height(layout.innerHeight).Render(rightContent)

	var divider string
	if layout.dividerWidth > 0 {
		divider = m.styles.divider.Render(verticalRule(layout.contentHeight))
	}

	top := lipgloss.JoinHorizontal(lipgloss.Top, leftView, divider, rightView)
	statusText := ansi.Truncate(m.statusLine(), contentWidth, "...")
	status := m.styles.statusBar.Width(contentWidth).Render(statusText)

	content := lipgloss.JoinVertical(lipgloss.Left, top, status)
	return m.styles.outer.Width(m.width).Height(m.height).Render(content)
}

func (m model) viewConfirm() string {
	contentWidth, contentHeight := contentSize(m.width, m.height)
	if m.confirmLoading {
		box := m.styles.modal.Render("Scanning directories...")
		placed := lipgloss.Place(contentWidth, contentHeight, lipgloss.Center, lipgloss.Center, box)
		return m.styles.outer.Width(m.width).Height(m.height).Render(placed)
	}
	if m.confirmErr != nil {
		box := m.styles.modal.Render(fmt.Sprintf("Error: %v\n\nEsc: Back", m.confirmErr))
		placed := lipgloss.Place(contentWidth, contentHeight, lipgloss.Center, lipgloss.Center, box)
		return m.styles.outer.Width(m.width).Height(m.height).Render(placed)
	}

	lines := []string{
		m.styles.modalTitle.Render("Confirm Selection"),
		"",
		fmt.Sprintf("Directories: %d", m.confirmResult.DirCount),
		fmt.Sprintf("Files: %d", m.confirmResult.FileCount),
		fmt.Sprintf("Total files: %d", m.confirmResult.TotalFiles),
		"",
		"Output dirs:",
	}
	for _, d := range m.confirmResult.OutputDirs {
		lines = append(lines, "- "+d)
	}
	lines = append(lines, "", "Enter: Continue  Esc: Back")
	box := m.styles.modal.Render(strings.Join(lines, "\n"))
	placed := lipgloss.Place(contentWidth, contentHeight, lipgloss.Center, lipgloss.Center, box)
	return m.styles.outer.Width(m.width).Height(m.height).Render(placed)
}

func (m model) viewConfig() string {
	contentWidth, contentHeight := contentSize(m.width, m.height)
	lines := []string{
		m.styles.modalTitle.Render("Configuration"),
		"",
		"Output Directory:",
		m.outputInput.View(),
		"",
		"Mode: Video Sticker",
		"",
		"Enter: Start  Esc: Back",
	}
	box := m.styles.modal.Render(strings.Join(lines, "\n"))
	placed := lipgloss.Place(contentWidth, contentHeight, lipgloss.Center, lipgloss.Center, box)
	return m.styles.outer.Width(m.width).Height(m.height).Render(placed)
}

func (m model) viewProcessing() string {
	contentWidth, contentHeight := contentSize(m.width, m.height)
	box := m.styles.modal.Render("Processing...")
	placed := lipgloss.Place(contentWidth, contentHeight, lipgloss.Center, lipgloss.Center, box)
	return m.styles.outer.Width(m.width).Height(m.height).Render(placed)
}

func (m model) viewSummary() string {
	contentWidth, contentHeight := contentSize(m.width, m.height)
	success := 0
	failed := 0
	for _, r := range m.results {
		if r.Err != nil || len(r.Issues) > 0 {
			failed++
			continue
		}
		success++
	}
	lines := []string{
		m.styles.modalTitle.Render("Summary"),
		"",
		fmt.Sprintf("Success: %d", success),
		fmt.Sprintf("Failed: %d", failed),
		"",
		"Press q to quit",
	}
	box := m.styles.modal.Render(strings.Join(lines, "\n"))
	placed := lipgloss.Place(contentWidth, contentHeight, lipgloss.Center, lipgloss.Center, box)
	return m.styles.outer.Width(m.width).Height(m.height).Render(placed)
}

func (m model) statusLine() string {
	focus := "Left"
	if m.focus == focusRight {
		focus = "Right"
	}
	parts := []string{
		fmt.Sprintf("Focus %s", focus),
		fmt.Sprintf("Selected %d", len(m.selectedList)),
		"Tab Switch  Enter Open/Next  Space Toggle  Backspace Remove  q Quit",
	}
	if m.status != "" {
		parts = append([]string{m.status}, parts...)
	}
	return strings.Join(parts, " | ")
}

func leftHeaderLine(filterView string, width int) string {
	line := fmt.Sprintf("%s%s", searchPrefix, filterView)
	if width <= 0 {
		return ""
	}
	return ansi.Truncate(line, width, "...")
}

func rightHeaderLine(count int) string {
	return fmt.Sprintf("Selected: %d", count)
}

func verticalRule(height int) string {
	if height < 1 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("|\n", height), "\n")
}

func loadDirCmd(path string, lister DirLister) tea.Cmd {
	return func() tea.Msg {
		entries, err := lister.ListDirEntries(path)
		return dirEntriesMsg{path: path, entries: entries, err: err}
	}
}

func expandCmd(expander SelectionExpander, selections []app.SelectionItem, outputDir string) tea.Cmd {
	return func() tea.Msg {
		result, err := expander.Expand(selections, outputDir)
		return confirmMsg{result: result, err: err}
	}
}

func runPipelineCmd(pipeline app.Pipeline, expander SelectionExpander, selections []app.SelectionItem, outputDir string) tea.Cmd {
	return func() tea.Msg {
		result, err := expander.Expand(selections, outputDir)
		if err != nil {
			return errMsg{err: err}
		}
		if len(result.Jobs) == 0 {
			return errMsg{err: fmt.Errorf("no valid inputs")}
		}
		results := pipeline.Run(context.Background(), result.Jobs)
		return doneMsg{results: results}
	}
}

func buildEntries(cwd string, entries []infra.DirEntry, selected map[string]struct{}) []entryItem {
	items := make([]entryItem, 0)
	parent := filepath.Dir(cwd)
	if parent != cwd {
		items = append(items, entryItem{path: parent, name: "..", isDir: true, isParent: true})
	}

	dirs := make([]entryItem, 0)
	files := make([]entryItem, 0)
	for _, e := range entries {
		if e.IsDir {
			dirs = append(dirs, entryItem{path: e.Path, name: e.Name, isDir: true})
			continue
		}
		if _, err := domain.DetectInputKind(e.Path); err != nil {
			continue
		}
		files = append(files, entryItem{path: e.Path, name: e.Name, isDir: false})
	}

	sort.Slice(dirs, func(i, j int) bool { return dirs[i].name < dirs[j].name })
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })

	items = append(items, dirs...)
	items = append(items, files...)

	for i := range items {
		if _, ok := selected[items[i].path]; ok {
			items[i].selected = true
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

func removeLastRune(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	return string(runes[:len(runes)-1])
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type leftListItem struct {
	entry entryItem
}

func (i leftListItem) Title() string {
	mark := "[ ]"
	if i.entry.selected {
		mark = "[x]"
	}
	name := i.entry.name
	if i.entry.isParent {
		name = "../"
	} else if i.entry.isDir {
		name += "/"
	}
	return fmt.Sprintf("%s %s", mark, name)
}

func (i leftListItem) Description() string { return "" }
func (i leftListItem) FilterValue() string { return i.entry.name }

type rightListItem struct {
	selection app.SelectionItem
	display   string
}

func (i rightListItem) Title() string       { return i.display }
func (i rightListItem) Description() string { return "" }
func (i rightListItem) FilterValue() string { return i.display }
