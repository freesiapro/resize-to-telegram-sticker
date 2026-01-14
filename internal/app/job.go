package app

import "github.com/resize-to-telegram-sticker/internal/domain"

type Job struct {
	InputPath string
	Kind      domain.InputKind
}

type Skipped struct {
	Path   string
	Reason string
}
