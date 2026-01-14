package domain

import "testing"

func TestValidateOutput(t *testing.T) {
	info := MediaInfo{Width: 512, Height: 256, FPS: 30, DurationSeconds: 3.0, HasAudio: false, CodecName: "vp9", FormatName: "webm"}
	issues := ValidateOutput(info, 200*1024)
	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %v", issues)
	}
}
