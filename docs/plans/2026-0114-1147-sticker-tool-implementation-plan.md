# Sticker Tool Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 用 TUI 支持视频/图片/GIF 输入并产出符合 Telegram 规格的视频贴纸（WebM/VP9、<=3s、<=30fps、512 边约束、<=256KB）。

**Architecture:** UI（Bubble Tea）-> App（用例编排）-> Domain（规格/策略/校验）-> Infra（ffmpeg/ffprobe/文件系统），仅通过接口依赖倒置。

**Tech Stack:** Go、Bubble Tea、Bubbles、u2takey/ffmpeg-go、ffmpeg/ffprobe、Go testing。

**Skills:** @superpowers:test-driven-development @superpowers:verification-before-completion

---

### Task 1: 添加依赖与目录结构

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`
- Create: `internal/domain/` `internal/app/` `internal/infra/` `internal/ui/`

**Step 1: 获取依赖**

Run: `go get github.com/charmbracelet/bubbletea github.com/charmbracelet/bubbles github.com/u2takey/ffmpeg-go`
Expected: go.mod/go.sum 更新

**Step 2: 整理模块**

Run: `go mod tidy`
Expected: go.sum 更新完成

**Step 3: 创建目录**

Run: `mkdir -p internal/domain internal/app internal/infra internal/ui`
Expected: 目录已创建

**Step 4: 基线测试**

Run: `go test ./...`
Expected: PASS (no test files yet)

**Step 5: Commit**

```bash
git add go.mod go.sum internal
git commit -m "chore: add dependencies and base structure"
```

---

### Task 2: Domain 约束与尺寸缩放

**Files:**
- Create: `internal/domain/constraints.go`
- Create: `internal/domain/constraints_test.go`

**Step 1: 写失败测试**

```go
package domain

import "testing"

func TestScaleToFit(t *testing.T) {
	cases := []struct {
		name string
		src  Size
		want Size
	}{
		{"wider", Size{Width: 1000, Height: 500}, Size{Width: 512, Height: 256}},
		{"taller", Size{Width: 500, Height: 1000}, Size{Width: 256, Height: 512}},
		{"square", Size{Width: 512, Height: 512}, Size{Width: 512, Height: 512}},
	}

	for _, c := range cases {
		got, err := ScaleToFit(c.src, MaxStickerSide)
		if err != nil {
			t.Fatalf("%s: unexpected err: %v", c.name, err)
		}
		if got != c.want {
			t.Fatalf("%s: got=%+v want=%+v", c.name, got, c.want)
		}
	}
}

func TestScaleToFitInvalid(t *testing.T) {
	_, err := ScaleToFit(Size{Width: 0, Height: 10}, MaxStickerSide)
	if err == nil {
		t.Fatal("expected error")
	}
}
```

**Step 2: 运行测试**

Run: `go test ./internal/domain -v`
Expected: FAIL (ScaleToFit 未实现)

**Step 3: 实现最小代码**

```go
package domain

import "fmt"

const (
	MaxStickerSide            = 512
	MaxStickerFPS             = 30
	MaxStickerDurationSeconds = 3
	MaxStickerSizeBytes       = 256 * 1024
	DefaultImageFPS           = 30
	DefaultImageDuration      = 3
)

type Size struct {
	Width  int
	Height int
}

func ScaleToFit(src Size, maxSide int) (Size, error) {
	if src.Width <= 0 || src.Height <= 0 {
		return Size{}, fmt.Errorf("invalid size: %dx%d", src.Width, src.Height)
	}

	if src.Width == src.Height {
		return Size{Width: maxSide, Height: maxSide}, nil
	}

	if src.Width > src.Height {
		return Size{
			Width:  maxSide,
			Height: int(float64(src.Height) * float64(maxSide) / float64(src.Width)),
		}, nil
	}

	return Size{
		Width:  int(float64(src.Width) * float64(maxSide) / float64(src.Height)),
		Height: maxSide,
	}, nil
}
```

**Step 4: 运行测试**

Run: `go test ./internal/domain -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/domain/constraints.go internal/domain/constraints_test.go
git commit -m "feat(domain): add constraints and scale"
```

---

### Task 3: 输入类型识别与媒体信息结构

**Files:**
- Create: `internal/domain/media.go`
- Create: `internal/domain/media_test.go`

**Step 1: 写失败测试**

```go
package domain

import "testing"

func TestDetectInputKind(t *testing.T) {
	cases := []struct {
		path string
		want InputKind
	}{
		{"a.mp4", InputKindVideo},
		{"a.png", InputKindImage},
		{"a.GIF", InputKindGIF},
	}

	for _, c := range cases {
		got, err := DetectInputKind(c.path)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if got != c.want {
			t.Fatalf("path=%s got=%s want=%s", c.path, got, c.want)
		}
	}
}

func TestDetectInputKindUnsupported(t *testing.T) {
	_, err := DetectInputKind("a.txt")
	if err == nil {
		t.Fatal("expected error")
	}
}
```

**Step 2: 运行测试**

Run: `go test ./internal/domain -v`
Expected: FAIL (DetectInputKind 未实现)

**Step 3: 实现最小代码**

```go
package domain

import (
	"fmt"
	"path/filepath"
	"strings"
)

type InputKind string

const (
	InputKindVideo InputKind = "video"
	InputKindImage InputKind = "image"
	InputKindGIF   InputKind = "gif"
)

type MediaInfo struct {
	Width           int
	Height          int
	FPS             float64
	DurationSeconds float64
	HasAudio        bool
	FormatName      string
	CodecName       string
}

var (
	videoExts = map[string]struct{}{".mp4": {}, ".mov": {}, ".webm": {}, ".mkv": {}, ".avi": {}}
	imageExts = map[string]struct{}{".png": {}, ".jpg": {}, ".jpeg": {}, ".webp": {}}
	gifExts   = map[string]struct{}{".gif": {}}
)

func DetectInputKind(path string) (InputKind, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if _, ok := gifExts[ext]; ok {
		return InputKindGIF, nil
	}
	if _, ok := imageExts[ext]; ok {
		return InputKindImage, nil
	}
	if _, ok := videoExts[ext]; ok {
		return InputKindVideo, nil
	}
	return "", fmt.Errorf("unsupported input: %s", path)
}
```

**Step 4: 运行测试**

Run: `go test ./internal/domain -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/domain/media.go internal/domain/media_test.go
git commit -m "feat(domain): add input kind detection"
```

---

### Task 4: 编码策略与输出校验

**Files:**
- Create: `internal/domain/strategy.go`
- Create: `internal/domain/strategy_test.go`
- Create: `internal/domain/validator.go`
- Create: `internal/domain/validator_test.go`

**Step 1: 写失败测试**

```go
package domain

import "testing"

func TestBuildAttemptsOrder(t *testing.T) {
	info := MediaInfo{Width: 1000, Height: 500, FPS: 60, DurationSeconds: 2.5}
	attempts, err := BuildAttempts(info, InputKindVideo)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(attempts) == 0 {
		t.Fatal("expected attempts")
	}
	if attempts[0].Width != 512 || attempts[0].Height != 256 {
		t.Fatalf("unexpected size: %+v", attempts[0])
	}
	if attempts[0].FPS != 30 {
		t.Fatalf("unexpected fps: %d", attempts[0].FPS)
	}
}

func TestValidateOutput(t *testing.T) {
	info := MediaInfo{Width: 512, Height: 256, FPS: 30, DurationSeconds: 3.0, HasAudio: false, CodecName: "vp9", FormatName: "webm"}
	issues := ValidateOutput(info, 200*1024)
	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %v", issues)
	}
}
```

**Step 2: 运行测试**

Run: `go test ./internal/domain -v`
Expected: FAIL (BuildAttempts/ValidateOutput 未实现)

**Step 3: 实现最小代码**

```go
package domain

import "math"

type EncodeAttempt struct {
	Width           int
	Height          int
	FPS             int
	BitrateKbps     int
	DurationSeconds int
	InputKind       InputKind
	LoopSeconds     int
}

func BuildAttempts(info MediaInfo, kind InputKind) ([]EncodeAttempt, error) {
	scaled, err := ScaleToFit(Size{Width: info.Width, Height: info.Height}, MaxStickerSide)
	if err != nil {
		return nil, err
	}

	baseFPS := int(math.Min(info.FPS, MaxStickerFPS))
	if baseFPS <= 0 {
		baseFPS = MaxStickerFPS
	}

	baseDuration := MaxStickerDurationSeconds
	if info.DurationSeconds > 0 && info.DurationSeconds < float64(MaxStickerDurationSeconds) {
		baseDuration = int(math.Ceil(info.DurationSeconds))
	}

	if kind == InputKindImage {
		baseFPS = DefaultImageFPS
		baseDuration = DefaultImageDuration
	}

	bitrateBase := int(float64(MaxStickerSizeBytes*8) / float64(baseDuration) / 1000.0)
	if bitrateBase < 150 {
		bitrateBase = 150
	}

	bitrateSteps := []float64{1.0, 0.85, 0.7, 0.55}
	scaleSteps := []float64{1.0, 0.9, 0.8}
	fpsSteps := []int{baseFPS, 24, 20, 15}

	attempts := make([]EncodeAttempt, 0)
	for _, b := range bitrateSteps {
		attempts = append(attempts, EncodeAttempt{
			Width:           scaled.Width,
			Height:          scaled.Height,
			FPS:             baseFPS,
			BitrateKbps:     int(float64(bitrateBase) * b),
			DurationSeconds: baseDuration,
			InputKind:       kind,
			LoopSeconds:     DefaultImageDuration,
		})
	}

	for _, s := range scaleSteps[1:] {
		w := int(float64(scaled.Width) * s)
		h := int(float64(scaled.Height) * s)
		for _, b := range bitrateSteps {
			attempts = append(attempts, EncodeAttempt{
				Width:           w,
				Height:          h,
				FPS:             baseFPS,
				BitrateKbps:     int(float64(bitrateBase) * b),
				DurationSeconds: baseDuration,
				InputKind:       kind,
				LoopSeconds:     DefaultImageDuration,
			})
		}
	}

	for _, f := range fpsSteps[1:] {
		for _, b := range bitrateSteps {
			attempts = append(attempts, EncodeAttempt{
				Width:           scaled.Width,
				Height:          scaled.Height,
				FPS:             f,
				BitrateKbps:     int(float64(bitrateBase) * b),
				DurationSeconds: baseDuration,
				InputKind:       kind,
				LoopSeconds:     DefaultImageDuration,
			})
		}
	}

	return attempts, nil
}
```

```go
package domain

import "strings"

type ValidationIssue struct {
	Code    string
	Message string
}

func ValidateOutput(info MediaInfo, sizeBytes int64) []ValidationIssue {
	issues := make([]ValidationIssue, 0)
	if sizeBytes > MaxStickerSizeBytes {
		issues = append(issues, ValidationIssue{Code: "size", Message: "size exceeds limit"})
	}
	if info.FPS > MaxStickerFPS {
		issues = append(issues, ValidationIssue{Code: "fps", Message: "fps exceeds limit"})
	}
	if info.DurationSeconds > float64(MaxStickerDurationSeconds) {
		issues = append(issues, ValidationIssue{Code: "duration", Message: "duration exceeds limit"})
	}
	if info.HasAudio {
		issues = append(issues, ValidationIssue{Code: "audio", Message: "audio stream present"})
	}
	if !strings.Contains(strings.ToLower(info.CodecName), "vp9") {
		issues = append(issues, ValidationIssue{Code: "codec", Message: "codec is not vp9"})
	}
	if !strings.Contains(strings.ToLower(info.FormatName), "webm") {
		issues = append(issues, ValidationIssue{Code: "format", Message: "format is not webm"})
	}

	if info.Width != MaxStickerSide && info.Height != MaxStickerSide {
		issues = append(issues, ValidationIssue{Code: "size", Message: "one side must be 512"})
	}
	if info.Width > MaxStickerSide || info.Height > MaxStickerSide {
		issues = append(issues, ValidationIssue{Code: "size", Message: "dimension exceeds 512"})
	}

	return issues
}
```

**Step 4: 运行测试**

Run: `go test ./internal/domain -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/domain/strategy.go internal/domain/strategy_test.go internal/domain/validator.go internal/domain/validator_test.go
git commit -m "feat(domain): add strategy and validation"
```

---

### Task 5: ffprobe 与文件扫描

**Files:**
- Create: `internal/infra/ffprobe.go`
- Create: `internal/infra/ffprobe_test.go`
- Create: `internal/infra/files.go`

**Step 1: 写失败测试**

```go
package infra

import (
	"testing"

	"github.com/resize-to-telegram-sticker/internal/domain"
)

func TestParseProbe(t *testing.T) {
	jsonStr := `{"streams":[{"codec_type":"video","width":512,"height":256,"r_frame_rate":"30/1","codec_name":"vp9"},{"codec_type":"audio"}],"format":{"format_name":"webm","duration":"2.9"}}`

	info, err := parseProbeJSON([]byte(jsonStr))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if info.Width != 512 || info.Height != 256 {
		t.Fatalf("unexpected size: %+v", info)
	}
	if info.HasAudio != true {
		t.Fatalf("expected audio")
	}
	if info.FPS != 30 {
		t.Fatalf("unexpected fps: %v", info.FPS)
	}
	if info.FormatName != "webm" || info.CodecName != "vp9" {
		t.Fatalf("unexpected format/codec: %+v", info)
	}
}

func TestParseFrameRate(t *testing.T) {
	fps := parseFrameRate("30000/1001")
	if fps < 29.9 || fps > 30.1 {
		t.Fatalf("unexpected fps: %v", fps)
	}
}
```

**Step 2: 运行测试**

Run: `go test ./internal/infra -v`
Expected: FAIL (parseProbeJSON 未实现)

**Step 3: 实现最小代码**

```go
package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/resize-to-telegram-sticker/internal/domain"
)

type FFprobeRunner struct {
	Path string
}

type probeJSON struct {
	Streams []struct {
		CodecType  string `json:"codec_type"`
		Width      int    `json:"width"`
		Height     int    `json:"height"`
		FrameRate  string `json:"r_frame_rate"`
		CodecName  string `json:"codec_name"`
		Duration   string `json:"duration"`
	} `json:"streams"`
	Format struct {
		FormatName string `json:"format_name"`
		Duration   string `json:"duration"`
	} `json:"format"`
}

func (r FFprobeRunner) Probe(ctx context.Context, path string) (domain.MediaInfo, error) {
	bin := r.Path
	if bin == "" {
		bin = "ffprobe"
	}
	args := []string{
		"-v", "error",
		"-show_entries", "stream=codec_type,width,height,r_frame_rate,codec_name,duration",
		"-show_entries", "format=duration,format_name",
		"-of", "json",
		path,
	}
	out, err := exec.CommandContext(ctx, bin, args...).Output()
	if err != nil {
		return domain.MediaInfo{}, fmt.Errorf("ffprobe failed: %w", err)
	}
	return parseProbeJSON(out)
}

func parseProbeJSON(data []byte) (domain.MediaInfo, error) {
	var p probeJSON
	if err := json.Unmarshal(data, &p); err != nil {
		return domain.MediaInfo{}, err
	}

	info := domain.MediaInfo{}
	for _, s := range p.Streams {
		if s.CodecType == "audio" {
			info.HasAudio = true
		}
		if s.CodecType == "video" {
			info.Width = s.Width
			info.Height = s.Height
			info.FPS = parseFrameRate(s.FrameRate)
			info.CodecName = s.CodecName
			if info.DurationSeconds == 0 {
				info.DurationSeconds = parseDuration(s.Duration)
			}
		}
	}
	if info.DurationSeconds == 0 {
		info.DurationSeconds = parseDuration(p.Format.Duration)
	}
	info.FormatName = p.Format.FormatName

	return info, nil
}

func parseFrameRate(v string) float64 {
	parts := strings.Split(v, "/")
	if len(parts) != 2 {
		return 0
	}
	num, _ := strconv.ParseFloat(parts[0], 64)
	den, _ := strconv.ParseFloat(parts[1], 64)
	if den == 0 {
		return 0
	}
	return num / den
}

func parseDuration(v string) float64 {
	f, _ := strconv.ParseFloat(v, 64)
	return f
}
```

```go
package infra

import (
	"io/fs"
	"path/filepath"
)

func ListFiles(root string) ([]string, error) {
	files := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files, err
}
```

**Step 4: 运行测试**

Run: `go test ./internal/infra -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/infra/ffprobe.go internal/infra/ffprobe_test.go internal/infra/files.go
git commit -m "feat(infra): add ffprobe and file scanner"
```

---

### Task 6: ffmpeg 编码执行器

**Files:**
- Create: `internal/infra/ffmpeg.go`

**Step 1: 实现编码器**

```go
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
	inputKw := ffmpeg.KwArgs{}
	if attempt.InputKind == domain.InputKindImage {
		inputKw["loop"] = 1
	}
	if attempt.InputKind == domain.InputKindGIF {
		inputKw["stream_loop"] = -1
	}

	stream := ffmpeg.Input(inputPath, inputKw)

	scaleArg := fmt.Sprintf("%d:%d", attempt.Width, attempt.Height)
	stream = stream.Filter("scale", ffmpeg.Args{scaleArg})

	if attempt.FPS > 0 {
		stream = stream.Filter("fps", ffmpeg.Args{fmt.Sprintf("%d", attempt.FPS)})
	}

	if opts.TrimSeconds > 0 {
		stream = stream.Trim(ffmpeg.KwArgs{"duration": fmt.Sprintf("%d", opts.TrimSeconds)})
	}

	outputKw := ffmpeg.KwArgs{
		"c:v": "libvpx-vp9",
		"b:v": fmt.Sprintf("%dk", attempt.BitrateKbps),
		"r":   fmt.Sprintf("%d", attempt.FPS),
		"t":   fmt.Sprintf("%d", attempt.DurationSeconds),
		"an":  "",
	}

	return stream.Output(outputPath, outputKw).OverWriteOutput().ErrorToStdOut().Run()
}
```

**Step 2: 基线测试**

Run: `go test ./...`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/infra/ffmpeg.go
git commit -m "feat(infra): add ffmpeg runner"
```

---

### Task 7: App Job 规划与 Pipeline

**Files:**
- Create: `internal/app/job.go`
- Create: `internal/app/job_planner.go`
- Create: `internal/app/job_planner_test.go`
- Create: `internal/app/pipeline.go`
- Create: `internal/app/pipeline_test.go`
- Create: `internal/app/events.go`

**Step 1: 写失败测试（JobPlanner）**

```go
package app

import (
	"testing"
)

func TestJobPlanner(t *testing.T) {
	planner := JobPlanner{}
	jobs, skipped := planner.Plan([]string{"a.mp4", "b.gif", "c.txt"})
	if len(jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(jobs))
	}
	if len(skipped) != 1 {
		t.Fatalf("expected 1 skipped, got %d", len(skipped))
	}
}
```

**Step 2: 写失败测试（Pipeline）**

```go
package app

import (
	"context"
	"errors"
	"testing"

	"github.com/resize-to-telegram-sticker/internal/domain"
)

type fakeProbe struct{}

func (fakeProbe) Probe(_ context.Context, _ string) (domain.MediaInfo, error) {
	return domain.MediaInfo{Width: 512, Height: 256, FPS: 30, DurationSeconds: 2.0}, nil
}

type fakeEncode struct{
	fail bool
}

func (f fakeEncode) Encode(_ context.Context, _ string, _ domain.EncodeAttempt, _ string, _ EncodeOptions) error {
	if f.fail {
		return errors.New("encode failed")
	}
	return nil
}

func TestPipelineContinuesOnError(t *testing.T) {
	p := Pipeline{
		Probe:  fakeProbe{},
		Encode: fakeEncode{fail: true},
	}

	jobs := []Job{{InputPath: "a.mp4"}, {InputPath: "b.mp4"}}
	results := p.Run(context.Background(), jobs)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}
```

**Step 3: 运行测试**

Run: `go test ./internal/app -v`
Expected: FAIL (JobPlanner/Pipeline 未实现)

**Step 4: 实现最小代码**

```go
package app

import "github.com/resize-to-telegram-sticker/internal/domain"

type Job struct {
	InputPath string
	Kind      domain.InputKind
}

type Skipped struct {
	Path   string
	Reason string
}
```

```go
package app

import "github.com/resize-to-telegram-sticker/internal/domain"

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
```

```go
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
```

```go
package app

type ProgressEvent struct {
	Total   int
	Done    int
	Current string
}
```

**Step 5: 运行测试**

Run: `go test ./internal/app -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/app

git commit -m "feat(app): add job planner and pipeline"
```

---

### Task 8: UI 与入口接线

**Files:**
- Create: `internal/ui/model.go`
- Create: `internal/ui/messages.go`
- Modify: `main.go`

**Step 1: 实现最小 UI**

```go
package ui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/resize-to-telegram-sticker/internal/app"
	"github.com/resize-to-telegram-sticker/internal/infra"
)

type inputMode string

const (
	modeFile inputMode = "file"
	modeDir  inputMode = "dir"
)

type model struct {
	modeList  list.Model
	pathInput textinput.Model
	spinner   spinner.Model
	progress  progress.Model

	state   string
	err     error
	results []app.Result

	planner  app.JobPlanner
	pipeline app.Pipeline
}

func NewModel() model {
	items := []list.Item{listItem{title: "File"}, listItem{title: "Directory"}}
	l := list.New(items, list.NewDefaultDelegate(), 20, 6)
	l.Title = "Input Mode"

	ti := textinput.New()
	ti.Placeholder = "/path/to/file/or/dir"
	ti.Focus()

	s := spinner.New()
	s.Spinner = spinner.Dot

	p := progress.New(progress.WithDefaultGradient())

	return model{
		modeList:  l,
		pathInput: ti,
		spinner:   s,
		progress:  p,
		state:     "select",
		planner:   app.JobPlanner{},
		pipeline: app.Pipeline{
			Probe:  infra.FFprobeRunner{},
			Encode: infra.FFmpegRunner{},
		},
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, textinput.Blink)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			if m.state == "select" {
				m.state = "run"
				return m, runPipelineCmd(m)
			}
		}
	case errMsg:
		m.state = "done"
		m.err = msg.err
		return m, nil
	case doneMsg:
		m.state = "done"
		m.results = msg.results
		return m, nil
	}

	var listCmd, inputCmd, spinCmd tea.Cmd
	m.modeList, listCmd = m.modeList.Update(msg)
	m.pathInput, inputCmd = m.pathInput.Update(msg)
	m.spinner, spinCmd = m.spinner.Update(msg)
	return m, tea.Batch(listCmd, inputCmd, spinCmd)
}

func (m model) View() string {
	switch m.state {
	case "select":
		return fmt.Sprintf("%s\n\n%s\n", m.modeList.View(), m.pathInput.View())
	case "run":
		return fmt.Sprintf("%s Processing...", m.spinner.View())
	case "done":
		if m.err != nil {
			return fmt.Sprintf("Error: %v", m.err)
		}
		return fmt.Sprintf("Done: %d result(s)", len(m.results))
	default:
		return ""
	}
}

func runPipelineCmd(m model) tea.Cmd {
	return func() tea.Msg {
		path := m.pathInput.Value()
		mode := modeFile
		if m.modeList.Index() == 1 {
			mode = modeDir
		}

		paths := []string{path}
		if mode == modeDir {
			files, err := infra.ListFiles(path)
			if err != nil {
				return errMsg{err: err}
			}
			paths = files
		}

		jobs, _ := m.planner.Plan(paths)
		if len(jobs) == 0 {
			return errMsg{err: fmt.Errorf("no valid inputs")}
		}

		results := m.pipeline.Run(context.Background(), jobs)
		return doneMsg{results: results}
	}
}
```

```go
package ui

import "github.com/resize-to-telegram-sticker/internal/app"

type errMsg struct {
	err error
}

type doneMsg struct {
	results []app.Result
}

type listItem struct {
	title string
}

func (i listItem) Title() string       { return i.title }
func (i listItem) Description() string { return "" }
func (i listItem) FilterValue() string { return i.title }
```

```go
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/resize-to-telegram-sticker/internal/ui"
)

func main() {
	p := tea.NewProgram(ui.NewModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("run failed: %v\n", err)
		os.Exit(1)
	}
}
```

**Step 2: 基线测试**

Run: `go test ./...`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/ui main.go
git commit -m "feat(ui): add minimal TUI flow"
```

---

Plan complete and saved to `docs/plans/2026-0114-1147-sticker-tool-implementation-plan.md`. Two execution options:

1. Subagent-Driven (this session) - I dispatch fresh subagent per task, review between tasks, fast iteration
2. Parallel Session (separate) - Open new session with executing-plans, batch execution with checkpoints

Which approach?
