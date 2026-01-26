package infra

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	ffmpeg "github.com/u2takey/ffmpeg-go"

	"github.com/freesiapro/resize-to-telegram-sticker/internal/domain"
)

type FFmpegRunner struct{}

func (r FFmpegRunner) Encode(ctx context.Context, inputPath string, attempt domain.EncodeAttempt, outputPath string, opts domain.EncodeOptions) error {
	inputKw := buildInputKwArgs(attempt)
	stream := ffmpeg.Input(inputPath, inputKw).Silent(true)
	stream.Context = ctx

	scaleArg := fmt.Sprintf("%d:%d", attempt.Width, attempt.Height)
	stream = stream.Filter("scale", ffmpeg.Args{scaleArg})

	if attempt.FPS > 0 {
		stream = stream.Filter("fps", ffmpeg.Args{fmt.Sprintf("%d", attempt.FPS)})
	}

	if opts.TrimSeconds > 0 {
		stream = stream.Trim(ffmpeg.KwArgs{"duration": fmt.Sprintf("%d", opts.TrimSeconds)})
	}

	outputKw := buildOutputKwArgs(attempt)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := stream.Output(outputPath, outputKw).
		OverWriteOutput().
		WithOutput(&stdout, &stderr).
		Run()

	if err != nil {
		stdoutText := stdout.String()
		stderrText := stderr.String()
		logPath, logErr := writeFFmpegErrorLog(outputPath, stdoutText, stderrText)
		suffix := formatFFmpegStderr(stderrText)
		if logErr == nil && logPath != "" {
			suffix = fmt.Sprintf("%s (ffmpeg log: %s)", suffix, logPath)
		} else if logErr != nil {
			suffix = fmt.Sprintf("%s (ffmpeg log write failed: %v)", suffix, logErr)
		}
		return fmt.Errorf("ffmpeg failed: %w%s", err, suffix)
	}
	return nil
}

func (r FFmpegRunner) EncodeImage(ctx context.Context, inputPath string, opts domain.ImageEncodeOptions, outputPath string) error {
	stream := ffmpeg.Input(inputPath).Silent(true)
	stream.Context = ctx

	scaleArg := buildImageScaleArg(opts.TargetSide)
	stream = stream.Filter("scale", ffmpeg.Args{scaleArg})
	if opts.PadToSquare {
		padArg := buildImagePadArg(opts.TargetSide)
		stream = stream.Filter("pad", ffmpeg.Args{padArg})
	}

	outputKw := buildImageOutputKwArgs()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := stream.Output(outputPath, outputKw).
		OverWriteOutput().
		WithOutput(&stdout, &stderr).
		Run()

	if err != nil {
		stdoutText := stdout.String()
		stderrText := stderr.String()
		logPath, logErr := writeFFmpegErrorLog(outputPath, stdoutText, stderrText)
		suffix := formatFFmpegStderr(stderrText)
		if logErr == nil && logPath != "" {
			suffix = fmt.Sprintf("%s (ffmpeg log: %s)", suffix, logPath)
		} else if logErr != nil {
			suffix = fmt.Sprintf("%s (ffmpeg log write failed: %v)", suffix, logErr)
		}
		return fmt.Errorf("ffmpeg failed: %w%s", err, suffix)
	}
	return nil
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

func buildImageScaleArg(targetSide int) string {
	return fmt.Sprintf("%d:%d:force_original_aspect_ratio=decrease", targetSide, targetSide)
}

func buildImagePadArg(targetSide int) string {
	return fmt.Sprintf("%d:%d:(ow-iw)/2:(oh-ih)/2:color=0x00000000", targetSide, targetSide)
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
	} else {
		kw["fps_mode"] = "vfr"
	}
	if attempt.DurationSeconds > 0 {
		kw["t"] = fmt.Sprintf("%d", attempt.DurationSeconds)
	}
	return kw
}

func buildImageOutputKwArgs() ffmpeg.KwArgs {
	return ffmpeg.KwArgs{
		"vframes": 1,
		"vcodec":  "png",
		"f":       "image2",
	}
}

func formatFFmpegStderr(stderr string) string {
	trimmed := strings.TrimSpace(stderr)
	if trimmed == "" {
		return ""
	}
	const maxLen = 2048
	if len(trimmed) > maxLen {
		trimmed = trimmed[len(trimmed)-maxLen:]
	}
	return ": " + trimmed
}

func writeFFmpegErrorLog(outputPath string, stdout string, stderr string) (string, error) {
	if outputPath == "" {
		return "", fmt.Errorf("empty output path")
	}
	logPath := outputPath + ".ffmpeg-error.log"
	content := formatFFmpegLog(stdout, stderr)
	if err := os.WriteFile(logPath, []byte(content), 0o644); err != nil {
		return "", err
	}
	return logPath, nil
}

func formatFFmpegLog(stdout string, stderr string) string {
	var builder strings.Builder
	writeSection := func(title string, content string) {
		builder.WriteString(title)
		builder.WriteString(":\n")
		trimmed := strings.TrimSpace(content)
		if trimmed == "" {
			builder.WriteString("<empty>\n")
			return
		}
		builder.WriteString(content)
		if !strings.HasSuffix(content, "\n") {
			builder.WriteString("\n")
		}
	}
	writeSection("STDOUT", stdout)
	writeSection("STDERR", stderr)
	return builder.String()
}
