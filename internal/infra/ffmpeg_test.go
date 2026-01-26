package infra

import (
	"testing"

	ffmpeg "github.com/u2takey/ffmpeg-go"

	"github.com/freesiapro/resize-to-telegram-sticker/internal/domain"
)

func TestBuildInputKwArgs(t *testing.T) {
	img := buildInputKwArgs(domain.EncodeAttempt{InputKind: domain.InputKindImage})
	if v, ok := img["loop"]; !ok || v != 1 {
		t.Fatalf("expected loop=1, got=%v", img)
	}

	gif := buildInputKwArgs(domain.EncodeAttempt{InputKind: domain.InputKindGIF})
	if v, ok := gif["stream_loop"]; !ok || v != -1 {
		t.Fatalf("expected stream_loop=-1, got=%v", gif)
	}

	vid := buildInputKwArgs(domain.EncodeAttempt{InputKind: domain.InputKindVideo})
	if len(vid) != 0 {
		t.Fatalf("expected empty args, got=%v", vid)
	}
}

func TestBuildOutputKwArgs(t *testing.T) {
	attempt := domain.EncodeAttempt{FPS: 30, BitrateKbps: 500, DurationSeconds: 3}
	got := buildOutputKwArgs(attempt)

	if got["c:v"] != "libvpx-vp9" {
		t.Fatalf("unexpected codec: %v", got["c:v"])
	}
	if got["b:v"] != "500k" {
		t.Fatalf("unexpected bitrate: %v", got["b:v"])
	}
	if got["r"] != "30" {
		t.Fatalf("unexpected fps: %v", got["r"])
	}
	if got["t"] != "3" {
		t.Fatalf("unexpected duration: %v", got["t"])
	}
	if _, ok := got["an"]; !ok {
		t.Fatalf("expected an flag")
	}

	_ = ffmpeg.KwArgs(got)
}

func TestBuildOutputKwArgsPreserveFPS(t *testing.T) {
	attempt := domain.EncodeAttempt{FPS: 0, BitrateKbps: 500, DurationSeconds: 3}
	got := buildOutputKwArgs(attempt)

	if _, ok := got["r"]; ok {
		t.Fatalf("unexpected fps override: %v", got["r"])
	}
	if got["fps_mode"] != "vfr" {
		t.Fatalf("unexpected fps_mode: %v", got["fps_mode"])
	}
}
