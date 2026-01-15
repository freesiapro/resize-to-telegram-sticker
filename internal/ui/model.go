package ui

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/resize-to-telegram-sticker/internal/app"
	"github.com/resize-to-telegram-sticker/internal/infra"
	"github.com/resize-to-telegram-sticker/internal/ui/core"
	"github.com/resize-to-telegram-sticker/internal/ui/screens"
)

type viewState int

const (
	stateBrowse viewState = iota
	stateConfirm
	stateConfig
	stateProcessing
	stateSummary
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

	width  int
	height int

	lister   DirLister
	expander SelectionExpander
	pipeline app.Pipeline

	styles core.Styles

	browse     screens.BrowseScreen
	confirm    screens.ConfirmScreen
	config     screens.ConfigScreen
	processing screens.ProcessingScreen
	summary    screens.SummaryScreen

	processingJobs    []app.Job
	processingResults []app.Result
}

func NewModel() model {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	return NewModelWithDeps(cwd, dirLister{}, app.SelectionExpander{})
}

func NewModelWithDeps(cwd string, lister DirLister, expander SelectionExpander) model {
	return model{
		state:    stateBrowse,
		lister:   lister,
		expander: expander,
		pipeline: app.Pipeline{
			Probe:  infra.FFprobeRunner{},
			Encode: infra.FFmpegRunner{},
		},
		styles:     core.NewStyles(),
		browse:     screens.NewBrowseScreen(cwd),
		confirm:    screens.ConfirmScreen{},
		config:     screens.NewConfigScreen(),
		processing: screens.NewProcessingScreen(),
		summary:    screens.SummaryScreen{},
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(loadDirCmd(m.browse.Cwd, m.lister), textinput.Blink)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.browse.Resize(m.width, m.height)
		return m, nil
	case core.DirEntriesMsg:
		if msg.Err != nil {
			m.browse.SetStatus(msg.Err)
			return m, nil
		}
		m.browse.ApplyDirEntries(msg.Path, msg.Entries)
		return m, nil
	case core.ConfirmMsg:
		m.confirm.SetResult(msg.Result, msg.Err)
		return m, nil
	case core.ErrMsg:
		m.browse.SetStatus(msg.Err)
		return m, nil
	case core.DoneMsg:
		m.state = stateSummary
		m.summary.SetResults(msg.Results)
		return m, nil
	case core.ProcessingPlanMsg:
		if msg.Err != nil {
			m.state = stateBrowse
			m.browse.SetStatus(msg.Err)
			return m, nil
		}
		if len(msg.Jobs) == 0 {
			m.state = stateBrowse
			m.browse.SetStatus(fmt.Errorf("no valid inputs"))
			return m, nil
		}
		m.processingJobs = msg.Jobs
		m.processingResults = make([]app.Result, len(msg.Jobs))
		m.processing.SetJobs(msg.Jobs)
		return m, m.startNextJob()
	case core.ProcessingJobResultMsg:
		if msg.Index >= 0 && msg.Index < len(m.processingResults) {
			m.processingResults[msg.Index] = msg.Result
		}
		m.processing.ApplyResult(msg.Index, msg.Result)
		if m.processing.DoneCount >= len(m.processingJobs) {
			m.state = stateSummary
			m.summary.SetResults(m.processingResults)
			return m, nil
		}
		return m, m.startNextJob()
	}

	if key, ok := msg.(tea.KeyMsg); ok {
		if key.String() == "ctrl+c" || key.String() == "q" {
			return m, tea.Quit
		}
	}

	switch m.state {
	case stateBrowse:
		updated, result := m.browse.Update(msg)
		m.browse = updated
		cmds := make([]tea.Cmd, 0, 2)
		if result.Cmd != nil {
			cmds = append(cmds, result.Cmd)
		}
		switch result.Event.Type {
		case screens.BrowseEventOpenDir:
			cmds = append(cmds, loadDirCmd(result.Event.Path, m.lister))
		case screens.BrowseEventStartConfirm:
			m.state = stateConfirm
			m.confirm.StartLoading()
			cmds = append(cmds, expandCmd(m.expander, m.browse.SelectedItems(), m.config.OutputDir))
		}
		return m, batchCmds(cmds)
	case stateConfirm:
		updated, result := m.confirm.Update(msg)
		m.confirm = updated
		switch result.Action {
		case screens.ConfirmActionBack:
			m.state = stateBrowse
		case screens.ConfirmActionContinue:
			m.state = stateConfig
			return m, m.config.Focus()
		}
		return m, result.Cmd
	case stateConfig:
		updated, result := m.config.Update(msg)
		m.config = updated
		switch result.Action {
		case screens.ConfigActionBack:
			m.state = stateBrowse
			return m, result.Cmd
		case screens.ConfigActionStart:
			m.state = stateProcessing
			m.processing.Reset()
			m.processingJobs = nil
			m.processingResults = nil
			return m, planProcessingCmd(m.expander, m.browse.SelectedItems(), m.config.OutputDir)
		}
		return m, result.Cmd
	case stateProcessing:
		return m, nil
	case stateSummary:
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	switch m.state {
	case stateConfirm:
		return m.confirm.View(m.width, m.height, m.styles)
	case stateConfig:
		return m.config.View(m.width, m.height, m.styles)
	case stateProcessing:
		return m.processing.View(m.width, m.height, m.styles)
	case stateSummary:
		return m.summary.View(m.width, m.height, m.styles)
	default:
		return m.browse.View(m.width, m.height, m.styles)
	}
}

func loadDirCmd(path string, lister DirLister) tea.Cmd {
	return func() tea.Msg {
		entries, err := lister.ListDirEntries(path)
		return core.DirEntriesMsg{Path: path, Entries: entries, Err: err}
	}
}

func expandCmd(expander SelectionExpander, selections []app.SelectionItem, outputDir string) tea.Cmd {
	return func() tea.Msg {
		result, err := expander.Expand(selections, outputDir)
		return core.ConfirmMsg{Result: result, Err: err}
	}
}

func planProcessingCmd(expander SelectionExpander, selections []app.SelectionItem, outputDir string) tea.Cmd {
	return func() tea.Msg {
		result, err := expander.Expand(selections, outputDir)
		if err != nil {
			return core.ProcessingPlanMsg{Err: err}
		}
		return core.ProcessingPlanMsg{Jobs: result.Jobs}
	}
}

func runJobCmd(pipeline app.Pipeline, job app.Job, index int) tea.Cmd {
	return func() tea.Msg {
		results := pipeline.Run(context.Background(), []app.Job{job})
		if len(results) == 0 {
			return core.ProcessingJobResultMsg{
				Index:  index,
				Result: app.Result{InputPath: job.InputPath, Err: fmt.Errorf("no result")},
			}
		}
		return core.ProcessingJobResultMsg{Index: index, Result: results[0]}
	}
}

func (m *model) startNextJob() tea.Cmd {
	next := m.processing.NextPendingIndex()
	if next < 0 {
		m.state = stateSummary
		m.summary.SetResults(m.processingResults)
		return nil
	}
	m.processing.MarkProcessing(next)
	return runJobCmd(m.pipeline, m.processingJobs[next], next)
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
