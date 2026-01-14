package domain

import "math"

type EncodeAttempt struct {
	Width           int
	Height          int
	FPS             int
	BitrateKbps     int
	DurationSeconds int
	InputKind       InputKind
	LoopSeconds     int
}

func BuildAttempts(info MediaInfo, kind InputKind) ([]EncodeAttempt, error) {
	scaled, err := ScaleToFit(Size{Width: info.Width, Height: info.Height}, MaxStickerSide)
	if err != nil {
		return nil, err
	}

	baseFPS := int(math.Min(info.FPS, float64(MaxStickerFPS)))
	if baseFPS <= 0 {
		baseFPS = MaxStickerFPS
	}

	baseDuration := MaxStickerDurationSeconds
	if info.DurationSeconds > 0 && info.DurationSeconds < float64(MaxStickerDurationSeconds) {
		baseDuration = int(math.Ceil(info.DurationSeconds))
	}

	if kind == InputKindImage {
		baseFPS = DefaultImageFPS
		baseDuration = DefaultImageDuration
	}
	if kind == InputKindGIF {
		if info.FPS > 0 {
			baseFPS = int(math.Min(info.FPS, float64(MaxStickerFPS)))
		} else {
			baseFPS = DefaultImageFPS
		}
		baseDuration = DefaultImageDuration
	}

	if baseDuration <= 0 {
		baseDuration = MaxStickerDurationSeconds
	}

	bitrateBase := int(float64(MaxStickerSizeBytes*8) / float64(baseDuration) / 1000.0)
	if bitrateBase < 150 {
		bitrateBase = 150
	}

	bitrateSteps := []float64{1.0, 0.85, 0.7, 0.55}
	scaleSteps := []float64{1.0, 0.9, 0.8}
	fpsSteps := []int{baseFPS, 24, 20, 15}

	loopSeconds := 0
	if kind == InputKindImage || kind == InputKindGIF {
		loopSeconds = DefaultImageDuration
	}

	attempts := make([]EncodeAttempt, 0)
	for _, b := range bitrateSteps {
		attempts = append(attempts, EncodeAttempt{
			Width:           scaled.Width,
			Height:          scaled.Height,
			FPS:             baseFPS,
			BitrateKbps:     int(float64(bitrateBase) * b),
			DurationSeconds: baseDuration,
			InputKind:       kind,
			LoopSeconds:     loopSeconds,
		})
	}

	for _, s := range scaleSteps[1:] {
		w := int(float64(scaled.Width) * s)
		h := int(float64(scaled.Height) * s)
		if w <= 0 {
			w = 1
		}
		if h <= 0 {
			h = 1
		}
		for _, b := range bitrateSteps {
			attempts = append(attempts, EncodeAttempt{
				Width:           w,
				Height:          h,
				FPS:             baseFPS,
				BitrateKbps:     int(float64(bitrateBase) * b),
				DurationSeconds: baseDuration,
				InputKind:       kind,
				LoopSeconds:     loopSeconds,
			})
		}
	}

	for _, f := range fpsSteps[1:] {
		if f <= 0 {
			continue
		}
		for _, b := range bitrateSteps {
			attempts = append(attempts, EncodeAttempt{
				Width:           scaled.Width,
				Height:          scaled.Height,
				FPS:             f,
				BitrateKbps:     int(float64(bitrateBase) * b),
				DurationSeconds: baseDuration,
				InputKind:       kind,
				LoopSeconds:     loopSeconds,
			})
		}
	}

	return attempts, nil
}
