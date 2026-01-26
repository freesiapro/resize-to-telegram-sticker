package task

import "github.com/freesiapro/resize-to-telegram-sticker/internal/domain"

type Result struct {
	InputPath  string
	OutputPath string
	Err        error
	Issues     []domain.ValidationIssue
}
