package cli

import (
	"github.com/freesiapro/resize-to-telegram-sticker/internal/app/handler"
	"github.com/freesiapro/resize-to-telegram-sticker/internal/app/pipeline"
	"github.com/freesiapro/resize-to-telegram-sticker/internal/app/target"
	"github.com/freesiapro/resize-to-telegram-sticker/internal/app/task"
	"github.com/freesiapro/resize-to-telegram-sticker/internal/infra"
)

func NewExecutor() task.Executor {
	videoPipeline := pipeline.Pipeline{
		Probe:  infra.FFprobeRunner{},
		Encode: infra.FFmpegRunner{},
	}
	imagePipeline := pipeline.ImagePipeline{Encode: infra.FFmpegRunner{}}

	return task.Executor{
		Handlers: map[task.TaskType]task.TaskHandler{
			task.TaskTypeVideoSticker: handler.VideoStickerHandler{Pipeline: videoPipeline},
			task.TaskTypeStaticSticker: handler.ImageStickerHandler{
				Pipeline: imagePipeline,
				Target:   target.TargetStaticSticker,
			},
			task.TaskTypeEmoji: handler.ImageStickerHandler{
				Pipeline: imagePipeline,
				Target:   target.TargetEmoji,
			},
		},
	}
}
