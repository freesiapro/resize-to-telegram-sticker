package selection

import (
	"sort"

	"github.com/freesiapro/resize-to-telegram-sticker/internal/app/job"
	"github.com/freesiapro/resize-to-telegram-sticker/internal/domain"
	"github.com/freesiapro/resize-to-telegram-sticker/internal/infra"
)

type SelectionItem struct {
	Path  string
	IsDir bool
}

type ExpandResult struct {
	Jobs       []job.Job
	DirCount   int
	FileCount  int
	TotalFiles int
	OutputDirs []string
	Skipped    []job.Skipped
}

type SelectionExpander struct {
	ListFiles func(root string) ([]string, error)
}

func (e SelectionExpander) Expand(selections []SelectionItem, outputDir string) (ExpandResult, error) {
	result := ExpandResult{}
	if outputDir == "" {
		outputDir = "./output"
	}
	listFiles := e.ListFiles
	if listFiles == nil {
		listFiles = infra.ListFiles
	}

	jobs := make([]job.Job, 0)
	seen := make(map[string]struct{})
	outputSet := make(map[string]struct{})

	files := make([]SelectionItem, 0)
	dirs := make([]SelectionItem, 0)
	for _, s := range selections {
		if s.IsDir {
			dirs = append(dirs, s)
			continue
		}
		files = append(files, s)
	}

	for _, s := range files {
		kind, err := domain.DetectInputKind(s.Path)
		if err != nil {
			result.Skipped = append(result.Skipped, job.Skipped{Path: s.Path, Reason: err.Error()})
			continue
		}
		if _, ok := seen[s.Path]; ok {
			continue
		}
		seen[s.Path] = struct{}{}
		jobs = append(jobs, job.Job{InputPath: s.Path, Kind: kind, OutputDir: outputDir})
		result.FileCount++
		result.TotalFiles++
		outputSet[outputDir] = struct{}{}
	}

	for _, s := range dirs {
		filesInDir, err := listFiles(s.Path)
		if err != nil {
			return ExpandResult{}, err
		}
		result.DirCount++
		for _, path := range filesInDir {
			kind, err := domain.DetectInputKind(path)
			if err != nil {
				result.Skipped = append(result.Skipped, job.Skipped{Path: path, Reason: err.Error()})
				continue
			}
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
			jobs = append(jobs, job.Job{InputPath: path, Kind: kind, OutputDir: outputDir})
			result.TotalFiles++
			outputSet[outputDir] = struct{}{}
		}
	}

	result.Jobs = jobs
	result.OutputDirs = sortedKeys(outputSet)
	return result, nil
}

func sortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
