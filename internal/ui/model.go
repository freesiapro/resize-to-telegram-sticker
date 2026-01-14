package ui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/resize-to-telegram-sticker/internal/app"
	"github.com/resize-to-telegram-sticker/internal/infra"
)

type inputMode string

const (
	modeFile inputMode = "file"
	modeDir  inputMode = "dir"
)

type model struct {
	modeList  list.Model
	pathInput textinput.Model
	spinner   spinner.Model
	progress  progress.Model

	state   string
	err     error
	results []app.Result

	planner  app.JobPlanner
	pipeline app.Pipeline
}

func NewModel() model {
	items := []list.Item{listItem{title: "File"}, listItem{title: "Directory"}}
	l := list.New(items, list.NewDefaultDelegate(), 20, 6)
	l.Title = "Input Mode"

	ti := textinput.New()
	ti.Placeholder = "/path/to/file/or/dir"
	ti.Focus()

	s := spinner.New()
	s.Spinner = spinner.Dot

	p := progress.New(progress.WithDefaultGradient())

	return model{
		modeList:  l,
		pathInput: ti,
		spinner:   s,
		progress:  p,
		state:     "select",
		planner:   app.JobPlanner{},
		pipeline: app.Pipeline{
			Probe:  infra.FFprobeRunner{},
			Encode: infra.FFmpegRunner{},
		},
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, textinput.Blink)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			if m.state == "select" {
				m.state = "run"
				return m, runPipelineCmd(m)
			}
		}
	case errMsg:
		m.state = "done"
		m.err = msg.err
		return m, nil
	case doneMsg:
		m.state = "done"
		m.results = msg.results
		return m, nil
	}

	var listCmd, inputCmd, spinCmd tea.Cmd
	m.modeList, listCmd = m.modeList.Update(msg)
	m.pathInput, inputCmd = m.pathInput.Update(msg)
	m.spinner, spinCmd = m.spinner.Update(msg)
	return m, tea.Batch(listCmd, inputCmd, spinCmd)
}

func (m model) View() string {
	switch m.state {
	case "select":
		return fmt.Sprintf("%s\n\n%s\n", m.modeList.View(), m.pathInput.View())
	case "run":
		return fmt.Sprintf("%s Processing...", m.spinner.View())
	case "done":
		if m.err != nil {
			return fmt.Sprintf("Error: %v", m.err)
		}
		return fmt.Sprintf("Done: %d result(s)", len(m.results))
	default:
		return ""
	}
}

func runPipelineCmd(m model) tea.Cmd {
	return func() tea.Msg {
		path := m.pathInput.Value()
		mode := modeFile
		if m.modeList.Index() == 1 {
			mode = modeDir
		}

		paths := []string{path}
		if mode == modeDir {
			files, err := infra.ListFiles(path)
			if err != nil {
				return errMsg{err: err}
			}
			paths = files
		}

		jobs, _ := m.planner.Plan(paths)
		if len(jobs) == 0 {
			return errMsg{err: fmt.Errorf("no valid inputs")}
		}

		results := m.pipeline.Run(context.Background(), jobs)
		return doneMsg{results: results}
	}
}
