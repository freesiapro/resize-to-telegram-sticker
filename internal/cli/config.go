package cli

import "github.com/freesiapro/resize-to-telegram-sticker/internal/app/target"

type WizardConfig struct {
	Target     target.TargetType
	InputPath  string
	InputIsDir bool
	OutputDir  string
}

type RunResult struct {
	Total     int
	Succeeded int
	Failed    int
}
