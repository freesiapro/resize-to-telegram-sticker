package target

import (
	"testing"

	"github.com/freesiapro/resize-to-telegram-sticker/internal/app/job"
	"github.com/freesiapro/resize-to-telegram-sticker/internal/domain"
)

func TestSummarizeJobs(t *testing.T) {
	jobs := []job.Job{
		{InputPath: "a.png", Kind: domain.InputKindImage},
		{InputPath: "b.gif", Kind: domain.InputKindGIF},
		{InputPath: "c.mp4", Kind: domain.InputKindVideo},
		{InputPath: "d.webp", Kind: domain.InputKindImage},
	}

	summary := SummarizeJobs(jobs)
	if summary.Total != 4 {
		t.Fatalf("total=%d", summary.Total)
	}
	if summary.Image != 2 {
		t.Fatalf("image=%d", summary.Image)
	}
	if summary.GIF != 1 {
		t.Fatalf("gif=%d", summary.GIF)
	}
	if summary.Video != 1 {
		t.Fatalf("video=%d", summary.Video)
	}
}

func TestEvaluateTargetStaticSticker(t *testing.T) {
	tests := []struct {
		name    string
		summary InputSummary
		status  TargetStatus
		message string
	}{
		{
			name:    "all images",
			summary: InputSummary{Total: 2, Image: 2},
			status:  TargetStatusOK,
		},
		{
			name:    "partial images",
			summary: InputSummary{Total: 3, Image: 1},
			status:  TargetStatusWarning,
			message: "Only images will be processed",
		},
		{
			name:    "no images",
			summary: InputSummary{Total: 2, Image: 0},
			status:  TargetStatusBlocked,
			message: "Must select images for this target",
		},
	}

	for _, tt := range tests {
		hint := EvaluateTarget(tt.summary, TargetStaticSticker)
		if hint.Status != tt.status {
			t.Fatalf("%s: status=%d", tt.name, hint.Status)
		}
		if hint.Message != tt.message {
			t.Fatalf("%s: message=%q", tt.name, hint.Message)
		}
	}
}

func TestEvaluateTargetEmoji(t *testing.T) {
	summary := InputSummary{Total: 2, Image: 1}
	hint := EvaluateTarget(summary, TargetEmoji)
	if hint.Status != TargetStatusWarning {
		t.Fatalf("status=%d", hint.Status)
	}
	if hint.Message != "Only images will be processed" {
		t.Fatalf("message=%q", hint.Message)
	}
}

func TestEvaluateTargetVideoSticker(t *testing.T) {
	tests := []struct {
		name    string
		summary InputSummary
		status  TargetStatus
		message string
	}{
		{
			name:    "all videos",
			summary: InputSummary{Total: 2, Video: 2},
			status:  TargetStatusOK,
		},
		{
			name:    "partial videos",
			summary: InputSummary{Total: 3, Video: 1, GIF: 1},
			status:  TargetStatusWarning,
			message: "Only videos or GIFs will be processed",
		},
		{
			name:    "no videos",
			summary: InputSummary{Total: 2, Image: 2},
			status:  TargetStatusBlocked,
			message: "Must select videos or GIFs for this target",
		},
	}

	for _, tt := range tests {
		hint := EvaluateTarget(tt.summary, TargetVideoSticker)
		if hint.Status != tt.status {
			t.Fatalf("%s: status=%d", tt.name, hint.Status)
		}
		if hint.Message != tt.message {
			t.Fatalf("%s: message=%q", tt.name, hint.Message)
		}
	}
}

func TestFilterJobsForTarget(t *testing.T) {
	jobs := []job.Job{
		{InputPath: "a.png", Kind: domain.InputKindImage},
		{InputPath: "b.gif", Kind: domain.InputKindGIF},
		{InputPath: "c.mp4", Kind: domain.InputKindVideo},
	}

	filteredVideo := FilterJobsForTarget(jobs, TargetVideoSticker)
	if len(filteredVideo) != 2 {
		t.Fatalf("video len=%d", len(filteredVideo))
	}

	filteredStatic := FilterJobsForTarget(jobs, TargetStaticSticker)
	if len(filteredStatic) != 1 {
		t.Fatalf("static len=%d", len(filteredStatic))
	}
}
