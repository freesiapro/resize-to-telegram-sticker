package pipeline

import (
	"context"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/freesiapro/resize-to-telegram-sticker/internal/app/job"
	"github.com/freesiapro/resize-to-telegram-sticker/internal/app/target"
	"github.com/freesiapro/resize-to-telegram-sticker/internal/app/task"
	"github.com/freesiapro/resize-to-telegram-sticker/internal/domain"
)

type ImageEncodeRunner interface {
	EncodeImage(ctx context.Context, inputPath string, opts domain.ImageEncodeOptions, outputPath string) error
}

type ImagePipeline struct {
	Encode ImageEncodeRunner
}

func (p ImagePipeline) Run(ctx context.Context, jobs []job.Job, targetType target.TargetType) []task.Result {
	results := make([]task.Result, 0, len(jobs))
	for _, job := range jobs {
		if err := ctx.Err(); err != nil {
			results = append(results, task.Result{InputPath: job.InputPath, Err: err})
			return results
		}
		if job.Kind != domain.InputKindImage {
			results = append(results, task.Result{InputPath: job.InputPath, Err: fmt.Errorf("unsupported input kind")})
			continue
		}
		if job.OutputDir != "" {
			if err := os.MkdirAll(job.OutputDir, 0o755); err != nil {
				results = append(results, task.Result{InputPath: job.InputPath, Err: err})
				continue
			}
		}

		output := imageOutputPath(job, targetType)
		opts, err := imageEncodeOptions(targetType)
		if err != nil {
			results = append(results, task.Result{InputPath: job.InputPath, Err: err})
			continue
		}

		if err := p.Encode.EncodeImage(ctx, job.InputPath, opts, output); err != nil {
			results = append(results, task.Result{InputPath: job.InputPath, Err: err})
			continue
		}

		info, err := probeImageInfo(output)
		if err != nil {
			results = append(results, task.Result{InputPath: job.InputPath, Err: err})
			continue
		}

		issues := validateImageOutput(info, targetType)
		if len(issues) == 0 {
			results = append(results, task.Result{InputPath: job.InputPath, OutputPath: output})
			continue
		}
		results = append(results, task.Result{InputPath: job.InputPath, Err: fmt.Errorf("validation failed"), Issues: issues})
	}
	return results
}

func imageOutputPath(job job.Job, targetType target.TargetType) string {
	baseName := strings.TrimSuffix(filepath.Base(job.InputPath), filepath.Ext(job.InputPath))
	suffix := "_sticker"
	if targetType == target.TargetEmoji {
		suffix = "_emoji"
	}
	name := baseName + suffix + ".png"
	if job.OutputDir == "" {
		return filepath.Join(filepath.Dir(job.InputPath), name)
	}
	return filepath.Join(job.OutputDir, name)
}

func imageEncodeOptions(targetType target.TargetType) (domain.ImageEncodeOptions, error) {
	switch targetType {
	case target.TargetStaticSticker:
		return domain.ImageEncodeOptions{TargetSide: domain.StaticStickerSide}, nil
	case target.TargetEmoji:
		return domain.ImageEncodeOptions{TargetSide: domain.EmojiSide, PadToSquare: true}, nil
	default:
		return domain.ImageEncodeOptions{}, fmt.Errorf("unsupported target")
	}
}

func validateImageOutput(info domain.ImageInfo, targetType target.TargetType) []domain.ValidationIssue {
	switch targetType {
	case target.TargetStaticSticker:
		return domain.ValidateStaticStickerImage(info)
	case target.TargetEmoji:
		return domain.ValidateEmojiImage(info)
	default:
		return []domain.ValidationIssue{{Code: "target", Message: "unsupported target"}}
	}
}

func probeImageInfo(path string) (domain.ImageInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return domain.ImageInfo{}, err
	}
	defer file.Close()

	config, format, err := image.DecodeConfig(file)
	if err != nil {
		return domain.ImageInfo{}, err
	}

	return domain.ImageInfo{Width: config.Width, Height: config.Height, Format: format}, nil
}
