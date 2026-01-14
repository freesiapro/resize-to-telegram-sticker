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
