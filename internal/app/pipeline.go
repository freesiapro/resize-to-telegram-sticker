package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/resize-to-telegram-sticker/internal/domain"
)

type ProbeRunner interface {
	Probe(ctx context.Context, path string) (domain.MediaInfo, error)
}

type EncodeOptions struct {
	TrimSeconds int
}

type EncodeRunner interface {
	Encode(ctx context.Context, inputPath string, attempt domain.EncodeAttempt, outputPath string, opts EncodeOptions) error
}

type Result struct {
	InputPath  string
	OutputPath string
	Err        error
	Issues     []domain.ValidationIssue
}

type Pipeline struct {
	Probe  ProbeRunner
	Encode EncodeRunner
}

func (p Pipeline) Run(ctx context.Context, jobs []Job) []Result {
	results := make([]Result, 0, len(jobs))
	for _, job := range jobs {
		info, err := p.Probe.Probe(ctx, job.InputPath)
		if err != nil {
			results = append(results, Result{InputPath: job.InputPath, Err: err})
			continue
		}

		attempts, err := domain.BuildAttempts(info, job.Kind)
		if err != nil {
			results = append(results, Result{InputPath: job.InputPath, Err: err})
			continue
		}

		output := outputPath(job.InputPath)
		var lastErr error
		var lastIssues []domain.ValidationIssue
		for _, a := range attempts {
			err = p.Encode.Encode(ctx, job.InputPath, a, output, EncodeOptions{TrimSeconds: domain.MaxStickerDurationSeconds})
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
				results = append(results, Result{InputPath: job.InputPath, OutputPath: output})
				lastErr = nil
				break
			}
			lastIssues = issues
			lastErr = fmt.Errorf("validation failed")
		}
		if lastErr != nil {
			results = append(results, Result{InputPath: job.InputPath, Err: lastErr, Issues: lastIssues})
		}
	}
	return results
}

func outputPath(input string) string {
	ext := filepath.Ext(input)
	base := input[:len(input)-len(ext)]
	return base + "_sticker.webm"
}
