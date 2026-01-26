package domain

import "testing"

func TestValidateStaticStickerImage(t *testing.T) {
	ok := ValidateStaticStickerImage(ImageInfo{Width: 512, Height: 300, Format: "png"})
	if len(ok) != 0 {
		t.Fatalf("expected no issues")
	}

	issues := ValidateStaticStickerImage(ImageInfo{Width: 300, Height: 300, Format: "png"})
	if len(issues) == 0 {
		t.Fatalf("expected issues")
	}

	formatIssues := ValidateStaticStickerImage(ImageInfo{Width: 512, Height: 512, Format: "WEBP"})
	if len(formatIssues) == 0 {
		t.Fatalf("expected format issue")
	}
}

func TestValidateEmojiImage(t *testing.T) {
	ok := ValidateEmojiImage(ImageInfo{Width: 100, Height: 100, Format: "PNG"})
	if len(ok) != 0 {
		t.Fatalf("expected no issues")
	}

	issues := ValidateEmojiImage(ImageInfo{Width: 100, Height: 90, Format: "png"})
	if len(issues) == 0 {
		t.Fatalf("expected issues")
	}
}
