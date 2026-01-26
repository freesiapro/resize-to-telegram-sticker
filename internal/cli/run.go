package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/huh/spinner"
	"github.com/samber/lo"

	"github.com/freesiapro/resize-to-telegram-sticker/internal/app/job"
	"github.com/freesiapro/resize-to-telegram-sticker/internal/app/selection"
	"github.com/freesiapro/resize-to-telegram-sticker/internal/app/target"
	"github.com/freesiapro/resize-to-telegram-sticker/internal/app/task"
)

type Plan struct {
	Config       WizardConfig
	ExpandResult selection.ExpandResult
	FilteredJobs []job.Job
}

func Run(ctx context.Context, out io.Writer) (RunResult, error) {
	accessible := os.Getenv("ACCESSIBLE") != ""
	expander := selection.SelectionExpander{}
	executor := NewExecutor()

	for {
		if err := ctx.Err(); err != nil {
			return RunResult{}, err
		}

		cfg, err := RunWizard(accessible)
		if err != nil {
			return RunResult{}, err
		}

		plan, err := buildPlan(accessible, expander, cfg)
		if err != nil {
			if msgErr := ShowMessage(accessible, "Invalid selection", err.Error()); msgErr != nil {
				return RunResult{}, msgErr
			}
			continue
		}

		summary := buildPlanSummary(plan)
		start, err := ConfirmPlan(accessible, "Confirm", summary)
		if err != nil {
			return RunResult{}, err
		}
		if !start {
			continue
		}

		tasks := buildTasks(plan.FilteredJobs, plan.Config.Target)
		return runTasks(ctx, out, executor, tasks)
	}
}

func buildPlan(accessible bool, expander selection.SelectionExpander, cfg WizardConfig) (Plan, error) {
	selectionItems := []selection.SelectionItem{{Path: cfg.InputPath, IsDir: cfg.InputIsDir}}

	var expanded selection.ExpandResult
	var expandErr error
	spinErr := spinner.New().
		Title("Scanning inputs...").
		Accessible(accessible).
		Action(func() {
			expanded, expandErr = expander.Expand(selectionItems, cfg.OutputDir)
		}).
		Run()
	if spinErr != nil {
		return Plan{}, spinErr
	}
	if expandErr != nil {
		return Plan{}, expandErr
	}

	filtered := target.FilterJobsForTarget(expanded.Jobs, cfg.Target)
	hint := target.EvaluateTarget(target.SummarizeJobs(expanded.Jobs), cfg.Target)
	if len(expanded.Jobs) == 0 && len(expanded.Skipped) > 0 {
		first := expanded.Skipped[0]
		return Plan{}, fmt.Errorf("no supported inputs (e.g. %s: %s)", first.Path, first.Reason)
	}
	if hint.Status == target.TargetStatusBlocked || len(filtered) == 0 {
		if hint.Message != "" {
			return Plan{}, fmt.Errorf("%s", hint.Message)
		}
		return Plan{}, fmt.Errorf("no valid inputs")
	}

	return Plan{
		Config:       cfg,
		ExpandResult: expanded,
		FilteredJobs: filtered,
	}, nil
}

func buildPlanSummary(plan Plan) string {
	lines := make([]string, 0, 32)

	lines = append(lines, fmt.Sprintf("Target: %s", target.TargetLabel(plan.Config.Target)))
	lines = append(lines, fmt.Sprintf("Input: %s", plan.Config.InputPath))
	lines = append(lines, fmt.Sprintf("Output: %s", plan.Config.OutputDir))
	lines = append(lines, "")

	lines = append(lines, fmt.Sprintf("Directories: %d", plan.ExpandResult.DirCount))
	lines = append(lines, fmt.Sprintf("Files: %d", plan.ExpandResult.FileCount))
	lines = append(lines, fmt.Sprintf("Total supported files: %d", plan.ExpandResult.TotalFiles))
	lines = append(lines, fmt.Sprintf("Tasks for target: %d", len(plan.FilteredJobs)))

	if len(plan.ExpandResult.Skipped) > 0 {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Skipped: %d", len(plan.ExpandResult.Skipped)))
		max := 8
		if len(plan.ExpandResult.Skipped) < max {
			max = len(plan.ExpandResult.Skipped)
		}
		for _, s := range plan.ExpandResult.Skipped[:max] {
			lines = append(lines, fmt.Sprintf("- %s (%s)", s.Path, s.Reason))
		}
		if len(plan.ExpandResult.Skipped) > max {
			lines = append(lines, "- ...")
		}
	}

	return strings.Join(lines, "\n")
}

func runTasks(ctx context.Context, out io.Writer, executor task.Executor, tasks []task.Task) (RunResult, error) {
	if len(tasks) == 0 {
		fmt.Fprintln(out, "No tasks to run.")
		return RunResult{}, nil
	}

	fmt.Fprintf(out, "Processing %d task(s). Press Ctrl+C to cancel.\n", len(tasks))

	events := make(chan task.TaskEvent, len(tasks)*2)
	doneCh := make(chan error, 1)
	go func() {
		doneCh <- executor.Run(ctx, tasks, events)
	}()

	results := make([]task.Result, len(tasks))
	doneCount := 0
	for event := range events {
		switch event.Type {
		case task.TaskStarted:
			fmt.Fprintf(out, "[RUN] %s\n", event.Task.Label)
		case task.TaskFinished:
			if event.Task.ID >= 0 && event.Task.ID < len(results) {
				results[event.Task.ID] = event.Result
			}
			doneCount++
			printResult(out, event.Result)
			fmt.Fprintf(out, "Done: %d/%d\n", doneCount, len(tasks))
		}
	}

	execErr := <-doneCh
	if execErr != nil {
		return RunResult{}, execErr
	}

	succeeded := lo.CountBy(results, func(r task.Result) bool {
		return r.Err == nil && len(r.Issues) == 0
	})
	failed := len(results) - succeeded

	fmt.Fprintln(out, "")
	fmt.Fprintf(out, "Summary: success=%d failed=%d\n", succeeded, failed)

	return RunResult{Total: len(results), Succeeded: succeeded, Failed: failed}, nil
}

func printResult(out io.Writer, result task.Result) {
	if result.Err == nil && len(result.Issues) == 0 {
		if result.OutputPath != "" {
			fmt.Fprintf(out, "[DONE] %s -> %s\n", result.InputPath, result.OutputPath)
		} else {
			fmt.Fprintf(out, "[DONE] %s\n", result.InputPath)
		}
		return
	}

	message := ""
	if result.Err != nil {
		message = result.Err.Error()
	} else if len(result.Issues) > 0 {
		message = result.Issues[0].Message
	}
	fmt.Fprintf(out, "[FAIL] %s (%s)\n", result.InputPath, message)
}

func buildTasks(jobs []job.Job, targetType target.TargetType) []task.Task {
	taskType := taskTypeForTarget(targetType)

	return lo.Map(jobs, func(j job.Job, index int) task.Task {
		return task.Task{
			ID:    index,
			Type:  taskType,
			Label: j.InputPath,
			Job:   j,
		}
	})
}

func taskTypeForTarget(targetType target.TargetType) task.TaskType {
	switch targetType {
	case target.TargetStaticSticker:
		return task.TaskTypeStaticSticker
	case target.TargetEmoji:
		return task.TaskTypeEmoji
	default:
		return task.TaskTypeVideoSticker
	}
}
