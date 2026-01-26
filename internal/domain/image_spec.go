package domain

import "strings"

type ImageInfo struct {
	Width  int
	Height int
	Format string
}

func ValidateStaticStickerImage(info ImageInfo) []ValidationIssue {
	issues := make([]ValidationIssue, 0)
	if !isPNG(info.Format) {
		issues = append(issues, ValidationIssue{Code: "format", Message: "format is not png"})
	}
	if info.Width != StaticStickerSide && info.Height != StaticStickerSide {
		issues = append(issues, ValidationIssue{Code: "size", Message: "one side must be 512"})
	}
	if info.Width > StaticStickerSide || info.Height > StaticStickerSide {
		issues = append(issues, ValidationIssue{Code: "size", Message: "dimension exceeds 512"})
	}
	return issues
}

func ValidateEmojiImage(info ImageInfo) []ValidationIssue {
	issues := make([]ValidationIssue, 0)
	if !isPNG(info.Format) {
		issues = append(issues, ValidationIssue{Code: "format", Message: "format is not png"})
	}
	if info.Width != EmojiSide || info.Height != EmojiSide {
		issues = append(issues, ValidationIssue{Code: "size", Message: "dimension must be 100x100"})
	}
	return issues
}

func isPNG(format string) bool {
	return strings.ToLower(format) == "png"
}
