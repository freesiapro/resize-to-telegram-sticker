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

func TestBuildAttemptsPreserveFPSWhenWithinLimit(t *testing.T) {
	info := MediaInfo{Width: 512, Height: 256, FPS: 25, DurationSeconds: 2.5}
	attempts, err := BuildAttempts(info, InputKindVideo)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(attempts) == 0 {
		t.Fatal("expected attempts")
	}
	if attempts[0].FPS != 0 {
		t.Fatalf("expected base attempt to preserve fps, got: %d", attempts[0].FPS)
	}
	for _, attempt := range attempts {
		if attempt.FPS == 25 {
			t.Fatalf("unexpected forced fps: %d", attempt.FPS)
		}
	}
}

func TestBuildAttemptsSkipFPSFallbackWhenUnknown(t *testing.T) {
	info := MediaInfo{Width: 512, Height: 256, FPS: 0, DurationSeconds: 2.5}
	attempts, err := BuildAttempts(info, InputKindVideo)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	for _, attempt := range attempts {
		if attempt.FPS != 0 {
			t.Fatalf("unexpected fps attempt: %d", attempt.FPS)
		}
	}
}

func TestBuildAttemptsAdaptiveBitrateStep(t *testing.T) {
	baseInfo := MediaInfo{Width: 1000, Height: 500, FPS: 30, DurationSeconds: 3}
	baseBitrate := expectedBaseBitrateKbps(3)
	cases := []struct {
		name           string
		inputSizeBytes int64
		bitrateBps     int64
		wantMultiplier float64
	}{
		{"uses-input-size", 1024 * 1024, 0, 0.55},
		{"uses-bitrate-when-input-missing", 0, 2_000_000, 0.55},
		{"fallback-when-no-size", 0, 0, 1.0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info := baseInfo
			info.InputSizeBytes = tc.inputSizeBytes
			info.BitrateBps = tc.bitrateBps
			attempts, err := BuildAttempts(info, InputKindVideo)
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if len(attempts) == 0 {
				t.Fatal("expected attempts")
			}
			want := int(float64(baseBitrate) * tc.wantMultiplier)
			if attempts[0].BitrateKbps != want {
				t.Fatalf("unexpected first bitrate: %d want %d", attempts[0].BitrateKbps, want)
			}
		})
	}
}

func expectedBaseBitrateKbps(durationSeconds int) int {
	bitrate := int(float64(MaxStickerSizeBytes*8) / float64(durationSeconds) / 1000.0)
	if bitrate < 150 {
		bitrate = 150
	}
	return bitrate
}
