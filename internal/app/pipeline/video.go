package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/freesiapro/resize-to-telegram-sticker/internal/app/job"
	"github.com/freesiapro/resize-to-telegram-sticker/internal/app/task"
	"github.com/freesiapro/resize-to-telegram-sticker/internal/domain"
)

type ProbeRunner interface {
	Probe(ctx context.Context, path string) (domain.MediaInfo, error)
}

type EncodeRunner interface {
	Encode(ctx context.Context, inputPath string, attempt domain.EncodeAttempt, outputPath string, opts domain.EncodeOptions) error
}

type Pipeline struct {
	Probe  ProbeRunner
	Encode EncodeRunner
}

func (p Pipeline) Run(ctx context.Context, jobs []job.Job) []task.Result {
	results := make([]task.Result, 0, len(jobs))
	for _, job := range jobs {
		if err := ctx.Err(); err != nil {
			results = append(results, task.Result{InputPath: job.InputPath, Err: err})
			return results
		}
		info, err := p.Probe.Probe(ctx, job.InputPath)
		if err != nil {
			results = append(results, task.Result{InputPath: job.InputPath, Err: err})
			continue
		}
		if stat, statErr := os.Stat(job.InputPath); statErr == nil {
			info.InputSizeBytes = stat.Size()
		}

		attempts, err := domain.BuildAttempts(info, job.Kind)
		if err != nil {
			results = append(results, task.Result{InputPath: job.InputPath, Err: err})
			continue
		}

		if job.OutputDir != "" {
			if err := os.MkdirAll(job.OutputDir, 0o755); err != nil {
				results = append(results, task.Result{InputPath: job.InputPath, Err: err})
				continue
			}
		}
		output := outputPath(job)
		var lastErr error
		var lastIssues []domain.ValidationIssue
		for _, a := range attempts {
			if err := ctx.Err(); err != nil {
				results = append(results, task.Result{InputPath: job.InputPath, Err: err})
				return results
			}
			err = p.Encode.Encode(ctx, job.InputPath, a, output, domain.EncodeOptions{TrimSeconds: domain.MaxStickerDurationSeconds})
			if err != nil {
				lastErr = err
				continue
			}

			stat, statErr := os.Stat(output)
			if statErr != nil {
				lastErr = statErr
				continue
			}

			outInfo, probeErr := p.Probe.Probe(ctx, output)
			if probeErr != nil {
				lastErr = probeErr
				continue
			}

			issues := domain.ValidateOutput(outInfo, stat.Size())
			if len(issues) == 0 {
				results = append(results, task.Result{InputPath: job.InputPath, OutputPath: output})
				lastErr = nil
				break
			}
			lastIssues = issues
			lastErr = fmt.Errorf("validation failed")
		}
		if lastErr != nil {
			results = append(results, task.Result{InputPath: job.InputPath, Err: lastErr, Issues: lastIssues})
		}
	}
	return results
}

func outputPath(job job.Job) string {
	baseName := strings.TrimSuffix(filepath.Base(job.InputPath), filepath.Ext(job.InputPath))
	name := baseName + "_sticker.webm"
	if job.OutputDir == "" {
		return filepath.Join(filepath.Dir(job.InputPath), name)
	}
	return filepath.Join(job.OutputDir, name)
}
