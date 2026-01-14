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

	cwd        string
	filterText string
	entries    []entryItem

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
	left := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	left.SetShowHelp(false)
	left.SetShowStatusBar(false)
	left.SetFilteringEnabled(false)
	left.DisableQuitKeybindings()

	right := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	right.SetShowHelp(false)
	right.SetShowStatusBar(false)
	right.SetFilteringEnabled(false)
	right.DisableQuitKeybindings()

	output := textinput.New()
	output.SetValue("./output")
	output.Placeholder = "./output"

	return model{
		state:        stateBrowse,
		focus:        focusLeft,
		cwd:          cwd,
		filterText:   "",
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
	return loadDirCmd(m.cwd, m.lister)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeLists()
		return m, nil
	case dirEntriesMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Error: %v", msg.err)
			return m, nil
		}
		m.cwd = msg.path
		m.entries = buildEntries(msg.path, msg.entries, m.selectedSet)
		m.refreshLeftList()
		return m, nil
	case confirmMsg:
		m.confirmLoading = false
		m.confirmErr = msg.err
		if msg.err == nil {
			m.confirmResult = msg.result
		}
		return m, nil
	case errMsg:
		m.status = fmt.Sprintf("Error: %v", msg.err)
		return m, nil
	case doneMsg:
		m.state = stateSummary
		m.results = msg.results
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
		switch m.state {
		case stateBrowse:
			return m.updateBrowse(msg)
		case stateConfirm:
			return m.updateConfirm(msg)
		case stateConfig:
			return m.updateConfig(msg)
		case stateSummary:
			return m, nil
		case stateProcessing:
			return m, nil
		}
	}

	return m, nil
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
		} else {
			m.focus = focusLeft
		}
		return m, nil
	case "enter":
		if m.focus == focusLeft {
			item, ok := m.leftList.SelectedItem().(leftListItem)
			if !ok || !item.entry.isDir {
				return m, nil
			}
			m.filterText = ""
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
		if m.focus == focusLeft {
			m.filterText = removeLastRune(m.filterText)
			m.refreshLeftList()
			return m, nil
		}
		if m.focus == focusRight {
			m.removeSelectedRight()
			return m, nil
		}
	}

	if m.focus == focusLeft && msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
		m.filterText += string(msg.Runes)
		m.refreshLeftList()
		return m, nil
	}

	return m.updateLists(msg)
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
	filtered := filterEntries(m.entries, m.filterText)
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
	statusHeight := 1
	headerHeight := 2
	borderHeight := 2

	paneHeight := m.height - statusHeight
	listHeight := paneHeight - headerHeight - borderHeight
	if listHeight < 1 {
		listHeight = 1
	}

	leftWidth := m.width / 2
	rightWidth := m.width - leftWidth
	contentWidthLeft := max(1, leftWidth-2)
	contentWidthRight := max(1, rightWidth-2)

	m.leftList.SetSize(contentWidthLeft, listHeight)
	m.rightList.SetSize(contentWidthRight, listHeight)
}

func (m model) viewBrowse() string {
	leftHeader := m.styles.header.Render(fmt.Sprintf("Current Dir: %s\nSearch: %s", m.cwd, m.filterText))
	rightHeader := m.styles.header.Render(fmt.Sprintf("Selected (%d)", len(m.selectedList)))

	leftContent := leftHeader + "\n" + m.leftList.View()
	rightContent := rightHeader + "\n" + m.rightList.View()

	leftPane := m.styles.paneBlurred
	rightPane := m.styles.paneBlurred
	if m.focus == focusLeft {
		leftPane = m.styles.paneFocused
	} else {
		rightPane = m.styles.paneFocused
	}

	leftWidth := m.width / 2
	rightWidth := m.width - leftWidth

	leftView := leftPane.Width(max(1, leftWidth)).Height(max(1, m.height-1)).Render(leftContent)
	rightView := rightPane.Width(max(1, rightWidth)).Height(max(1, m.height-1)).Render(rightContent)

	status := m.styles.statusBar.Render(m.statusLine())

	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, leftView, rightView),
		status,
	)
}

func (m model) viewConfirm() string {
	if m.confirmLoading {
		box := m.styles.modal.Render("Scanning directories...")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	}
	if m.confirmErr != nil {
		box := m.styles.modal.Render(fmt.Sprintf("Error: %v\n\nEsc: Back", m.confirmErr))
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
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
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m model) viewConfig() string {
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
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m model) viewProcessing() string {
	box := m.styles.modal.Render("Processing...")
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m model) viewSummary() string {
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
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m model) statusLine() string {
	focus := "Left"
	if m.focus == focusRight {
		focus = "Right"
	}
	parts := []string{
		fmt.Sprintf("Focus: %s", focus),
		fmt.Sprintf("Selected: %d", len(m.selectedList)),
		"Filter: dirs/images/gif/video",
		"Tab: switch  Enter: open/next  Space: toggle  q: quit",
	}
	if m.status != "" {
		parts = append([]string{m.status}, parts...)
	}
	return strings.Join(parts, " | ")
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
