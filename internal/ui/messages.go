package ui

import "github.com/resize-to-telegram-sticker/internal/app"

type errMsg struct {
	err error
}

type doneMsg struct {
	results []app.Result
}

type listItem struct {
	title string
}

func (i listItem) Title() string       { return i.title }
func (i listItem) Description() string { return "" }
func (i listItem) FilterValue() string { return i.title }
