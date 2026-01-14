package app

import "github.com/resize-to-telegram-sticker/internal/domain"

type Job struct {
	InputPath string
	Kind      domain.InputKind
	OutputDir string
}

type Skipped struct {
	Path   string
	Reason string
}
