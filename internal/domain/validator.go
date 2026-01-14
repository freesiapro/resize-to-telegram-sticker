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
