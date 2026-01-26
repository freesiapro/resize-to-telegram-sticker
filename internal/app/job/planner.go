package job

import "github.com/freesiapro/resize-to-telegram-sticker/internal/domain"

type JobPlanner struct{}

func (JobPlanner) Plan(paths []string) ([]Job, []Skipped) {
	jobs := make([]Job, 0)
	skipped := make([]Skipped, 0)

	for _, p := range paths {
		kind, err := domain.DetectInputKind(p)
		if err != nil {
			skipped = append(skipped, Skipped{Path: p, Reason: err.Error()})
			continue
		}
		jobs = append(jobs, Job{InputPath: p, Kind: kind})
	}

	return jobs, skipped
}
