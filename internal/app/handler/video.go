package handler

import (
	"context"
	"fmt"

	"github.com/freesiapro/resize-to-telegram-sticker/internal/app/job"
	"github.com/freesiapro/resize-to-telegram-sticker/internal/app/pipeline"
	"github.com/freesiapro/resize-to-telegram-sticker/internal/app/task"
)

type VideoStickerHandler struct {
	Pipeline pipeline.Pipeline
}

func (h VideoStickerHandler) Handle(ctx context.Context, taskItem task.Task) task.Result {
	results := h.Pipeline.Run(ctx, []job.Job{taskItem.Job})
	if len(results) == 0 {
		return task.Result{InputPath: taskItem.Job.InputPath, Err: fmt.Errorf("no result")}
	}
	return results[0]
}
