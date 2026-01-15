package core

import (
	"github.com/resize-to-telegram-sticker/internal/app"
	"github.com/resize-to-telegram-sticker/internal/infra"
)

type ErrMsg struct {
	Err error
}

type DoneMsg struct {
	Results []app.Result
}

type DirEntriesMsg struct {
	Path    string
	Entries []infra.DirEntry
	Err     error
}

type ConfirmMsg struct {
	Result app.ExpandResult
	Err    error
}

type ListItem struct {
	Value string
}

func (i ListItem) Title() string       { return i.Value }
func (i ListItem) Description() string { return "" }
func (i ListItem) FilterValue() string { return i.Value }
