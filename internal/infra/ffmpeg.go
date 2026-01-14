package infra

import (
	"context"
	"fmt"

	ffmpeg "github.com/u2takey/ffmpeg-go"

	"github.com/resize-to-telegram-sticker/internal/domain"
)

type FFmpegRunner struct{}

type EncodeOptions struct {
	TrimSeconds int
}

func (r FFmpegRunner) Encode(ctx context.Context, inputPath string, attempt domain.EncodeAttempt, outputPath string, opts EncodeOptions) error {
	inputKw := buildInputKwArgs(attempt)
	stream := ffmpeg.Input(inputPath, inputKw)

	scaleArg := fmt.Sprintf("%d:%d", attempt.Width, attempt.Height)
	stream = stream.Filter("scale", ffmpeg.Args{scaleArg})

	if attempt.FPS > 0 {
		stream = stream.Filter("fps", ffmpeg.Args{fmt.Sprintf("%d", attempt.FPS)})
	}

	if opts.TrimSeconds > 0 {
		stream = stream.Trim(ffmpeg.KwArgs{"duration": fmt.Sprintf("%d", opts.TrimSeconds)})
	}

	outputKw := buildOutputKwArgs(attempt)

	return stream.Output(outputPath, outputKw).OverWriteOutput().ErrorToStdOut().Run()
}

func buildInputKwArgs(attempt domain.EncodeAttempt) ffmpeg.KwArgs {
	kw := ffmpeg.KwArgs{}
	if attempt.InputKind == domain.InputKindImage {
		kw["loop"] = 1
	}
	if attempt.InputKind == domain.InputKindGIF {
		kw["stream_loop"] = -1
	}
	return kw
}

func buildOutputKwArgs(attempt domain.EncodeAttempt) ffmpeg.KwArgs {
	kw := ffmpeg.KwArgs{
		"c:v": "libvpx-vp9",
		"an":  "",
	}
	if attempt.BitrateKbps > 0 {
		kw["b:v"] = fmt.Sprintf("%dk", attempt.BitrateKbps)
	}
	if attempt.FPS > 0 {
		kw["r"] = fmt.Sprintf("%d", attempt.FPS)
	}
	if attempt.DurationSeconds > 0 {
		kw["t"] = fmt.Sprintf("%d", attempt.DurationSeconds)
	}
	return kw
}
