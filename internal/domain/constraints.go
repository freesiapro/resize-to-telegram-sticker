package domain

import "fmt"

const (
	MaxStickerSide            = 512
	MaxStickerFPS             = 30
	MaxStickerDurationSeconds = 3
	MaxStickerSizeBytes       = 256 * 1024
	DefaultImageFPS           = 30
	DefaultImageDuration      = 3
	StaticStickerSide         = 512
	EmojiSide                 = 100
)

type Size struct {
	Width  int
	Height int
}

func ScaleToFit(src Size, maxSide int) (Size, error) {
	if src.Width <= 0 || src.Height <= 0 {
		return Size{}, fmt.Errorf("invalid size: %dx%d", src.Width, src.Height)
	}

	if src.Width == src.Height {
		return Size{Width: maxSide, Height: maxSide}, nil
	}

	if src.Width > src.Height {
		height := int(float64(src.Height) * float64(maxSide) / float64(src.Width))
		if height <= 0 {
			height = 1
		}
		return Size{Width: maxSide, Height: height}, nil
	}

	width := int(float64(src.Width) * float64(maxSide) / float64(src.Height))
	if width <= 0 {
		width = 1
	}
	return Size{Width: width, Height: maxSide}, nil
}
