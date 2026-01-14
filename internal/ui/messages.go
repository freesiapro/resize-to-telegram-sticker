package ui

import (
	"github.com/resize-to-telegram-sticker/internal/app"
	"github.com/resize-to-telegram-sticker/internal/infra"
)

type errMsg struct {
	err error
}

type doneMsg struct {
	results []app.Result
}

type dirEntriesMsg struct {
	path    string
	entries []infra.DirEntry
	err     error
}

type confirmMsg struct {
	result app.ExpandResult
	err    error
}

type listItem struct {
	title string
}

func (i listItem) Title() string       { return i.title }
func (i listItem) Description() string { return "" }
func (i listItem) FilterValue() string { return i.title }
