package app

import (
	"context"
	"fmt"
)

type VideoStickerPayload struct {
	Job Job
}

type VideoStickerHandler struct {
	Pipeline Pipeline
}

func (h VideoStickerHandler) Handle(ctx context.Context, task Task) Result {
	payload, ok := task.Payload.(VideoStickerPayload)
	if !ok {
		return Result{InputPath: task.Label, Err: fmt.Errorf("invalid payload")}
	}
	results := h.Pipeline.Run(ctx, []Job{payload.Job})
	if len(results) == 0 {
		return Result{InputPath: payload.Job.InputPath, Err: fmt.Errorf("no result")}
	}
	return results[0]
}
