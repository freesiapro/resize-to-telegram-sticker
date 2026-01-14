package app

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/resize-to-telegram-sticker/internal/domain"
)

type fakeProbe struct{}

func (fakeProbe) Probe(_ context.Context, _ string) (domain.MediaInfo, error) {
	return domain.MediaInfo{Width: 512, Height: 256, FPS: 30, DurationSeconds: 2.0}, nil
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
		Probe:  fakeProbe{},
		Encode: fakeEncode{fail: true},
	}

	jobs := []Job{{InputPath: "a.mp4"}, {InputPath: "b.mp4"}}
	results := p.Run(context.Background(), jobs)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestPipelineUsesOutputDir(t *testing.T) {
	encoder := &captureEncode{}
	p := Pipeline{Probe: fakeProbe{}, Encode: encoder}
	jobs := []Job{{InputPath: "/tmp/a.gif", Kind: domain.InputKindGIF, OutputDir: "/tmp/out"}}
	_ = p.Run(context.Background(), jobs)
	if encoder.output == "" || filepath.Dir(encoder.output) != "/tmp/out" {
		t.Fatalf("unexpected output: %s", encoder.output)
	}
}
