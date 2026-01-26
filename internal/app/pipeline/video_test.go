package pipeline

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/freesiapro/resize-to-telegram-sticker/internal/app/job"
	"github.com/freesiapro/resize-to-telegram-sticker/internal/domain"
)

type fakeProbe struct {
	info domain.MediaInfo
	err  error
}

func (f fakeProbe) Probe(_ context.Context, _ string) (domain.MediaInfo, error) {
	if f.err != nil {
		return domain.MediaInfo{}, f.err
	}
	return f.info, nil
}

type fakeEncode struct {
	fail bool
}

func (f fakeEncode) Encode(_ context.Context, _ string, _ domain.EncodeAttempt, _ string, _ domain.EncodeOptions) error {
	if f.fail {
		return errors.New("encode failed")
	}
	return nil
}

type captureEncode struct{ output string }

func (c *captureEncode) Encode(_ context.Context, _ string, _ domain.EncodeAttempt, outputPath string, _ domain.EncodeOptions) error {
	c.output = outputPath
	return nil
}

func TestPipelineContinuesOnError(t *testing.T) {
	p := Pipeline{
		Probe:  fakeProbe{info: domain.MediaInfo{Width: 512, Height: 256, FPS: 30, DurationSeconds: 2.0}},
		Encode: fakeEncode{fail: true},
	}

	jobs := []job.Job{{InputPath: "a.mp4"}, {InputPath: "b.mp4"}}
	results := p.Run(context.Background(), jobs)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestPipelineUsesOutputDir(t *testing.T) {
	encoder := &captureEncode{}
	p := Pipeline{Probe: fakeProbe{info: domain.MediaInfo{Width: 512, Height: 256, FPS: 30, DurationSeconds: 2.0}}, Encode: encoder}
	jobs := []job.Job{{InputPath: "/tmp/a.gif", Kind: domain.InputKindGIF, OutputDir: "/tmp/out"}}
	_ = p.Run(context.Background(), jobs)
	if encoder.output == "" || filepath.Dir(encoder.output) != "/tmp/out" {
		t.Fatalf("unexpected output: %s", encoder.output)
	}
}

type captureFirstAttempt struct {
	first domain.EncodeAttempt
	seen  bool
}

func (c *captureFirstAttempt) Encode(_ context.Context, _ string, attempt domain.EncodeAttempt, _ string, _ domain.EncodeOptions) error {
	if !c.seen {
		c.first = attempt
		c.seen = true
	}
	return errors.New("encode failed")
}

func TestPipelineUsesInputSizeForFirstAttempt(t *testing.T) {
	tempFile, err := os.CreateTemp(t.TempDir(), "input-*.mp4")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	payload := make([]byte, 1024*1024)
	if _, err := tempFile.Write(payload); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	encoder := &captureFirstAttempt{}
	p := Pipeline{
		Probe:  fakeProbe{info: domain.MediaInfo{Width: 512, Height: 256, FPS: 30, DurationSeconds: 3}},
		Encode: encoder,
	}
	_ = p.Run(context.Background(), []job.Job{{InputPath: tempFile.Name(), Kind: domain.InputKindVideo}})

	baseBitrate := expectedBaseBitrateKbps(3)
	want := int(float64(baseBitrate) * 0.55)
	if encoder.first.BitrateKbps != want {
		t.Fatalf("unexpected first bitrate: %d want %d", encoder.first.BitrateKbps, want)
	}
}

func expectedBaseBitrateKbps(durationSeconds int) int {
	bitrate := int(float64(domain.MaxStickerSizeBytes*8) / float64(durationSeconds) / 1000.0)
	if bitrate < 150 {
		bitrate = 150
	}
	return bitrate
}
