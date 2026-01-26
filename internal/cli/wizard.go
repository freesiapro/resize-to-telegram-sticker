package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/freesiapro/resize-to-telegram-sticker/internal/app/target"
)

type inputMode string

const (
	inputModeFile inputMode = "file"
	inputModeDir  inputMode = "dir"
)

var supportedFileTypes = []string{
	".mp4",
	".mov",
	".webm",
	".mkv",
	".avi",
	".gif",
	".png",
	".jpg",
	".jpeg",
	".webp",
}

func RunWizard(accessible bool) (WizardConfig, error) {
	selectedTarget := target.TargetVideoSticker
	mode := inputModeFile

	var filePath string
	var dirPath string
	outputDir := "./output"

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[target.TargetType]().
				Title("Target").
				Options(
					huh.NewOption(target.TargetLabel(target.TargetVideoSticker), target.TargetVideoSticker),
					huh.NewOption(target.TargetLabel(target.TargetStaticSticker), target.TargetStaticSticker),
					huh.NewOption(target.TargetLabel(target.TargetEmoji), target.TargetEmoji),
				).
				Value(&selectedTarget),
		),
		huh.NewGroup(
			huh.NewSelect[inputMode]().
				Title("Input type").
				Options(
					huh.NewOption("File", inputModeFile),
					huh.NewOption("Directory", inputModeDir),
				).
				Value(&mode),
		),
		huh.NewGroup(
			huh.NewFilePicker().
				Title("Input file").
				Description("Select a file to process.").
				AllowedTypes(supportedFileTypes).
				CurrentDirectory(".").
				ShowHidden(false).
				ShowSize(true).
				FileAllowed(true).
				DirAllowed(false).
				Value(&filePath),
		).WithHideFunc(func() bool {
			return mode != inputModeFile
		}),
		huh.NewGroup(
			huh.NewFilePicker().
				Title("Input directory").
				Description("Select a directory to scan recursively.").
				CurrentDirectory(".").
				ShowHidden(false).
				ShowSize(false).
				FileAllowed(false).
				DirAllowed(true).
				Value(&dirPath),
		).WithHideFunc(func() bool {
			return mode != inputModeDir
		}),
		huh.NewGroup(
			huh.NewInput().
				Title("Output directory").
				Placeholder("./output").
				Value(&outputDir).
				Validate(huh.ValidateNotEmpty()),
		),
	).WithAccessible(accessible)

	if err := form.Run(); err != nil {
		return WizardConfig{}, err
	}

	cfg := WizardConfig{
		Target:     selectedTarget,
		OutputDir:  strings.TrimSpace(outputDir),
		InputIsDir: mode == inputModeDir,
	}
	if cfg.OutputDir == "" {
		cfg.OutputDir = "./output"
	}

	if cfg.InputIsDir {
		cfg.InputPath = strings.TrimSpace(dirPath)
	} else {
		cfg.InputPath = strings.TrimSpace(filePath)
	}
	if cfg.InputPath == "" {
		return WizardConfig{}, fmt.Errorf("input path is required")
	}

	return cfg, nil
}

func ConfirmPlan(accessible bool, title string, summary string) (bool, error) {
	confirmed := false
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title(title).
				Description(summary).
				Next(true).
				NextLabel("Continue"),
		),
		huh.NewGroup(
			huh.NewConfirm().
				Title("Start processing?").
				Affirmative("Start").
				Negative("Back").
				Value(&confirmed),
		),
	).WithAccessible(accessible)

	if err := form.Run(); err != nil {
		return false, err
	}
	return confirmed, nil
}

func ShowMessage(accessible bool, title string, summary string) error {
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title(title).
				Description(summary).
				Next(true).
				NextLabel("Back"),
		),
	).WithAccessible(accessible)
	return form.Run()
}
