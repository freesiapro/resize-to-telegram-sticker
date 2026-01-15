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

	baseAttemptFPS := pickBaseAttemptFPS(info, kind)
	fallbackBaseFPS, allowFPSFallback := pickFallbackBaseFPS(info, kind)
	fpsFallbackSteps := buildFPSFallbackSteps(fallbackBaseFPS, allowFPSFallback)

	baseDuration := MaxStickerDurationSeconds
	if info.DurationSeconds > 0 && info.DurationSeconds < float64(MaxStickerDurationSeconds) {
		baseDuration = int(math.Ceil(info.DurationSeconds))
	}

	if kind == InputKindImage {
		baseDuration = DefaultImageDuration
	}
	if kind == InputKindGIF {
		baseDuration = DefaultImageDuration
	}

	if baseDuration <= 0 {
		baseDuration = MaxStickerDurationSeconds
	}

	bitrateBase := int(float64(MaxStickerSizeBytes*8) / float64(baseDuration) / 1000.0)
	if bitrateBase < 150 {
		bitrateBase = 150
	}

	bitrateSteps := []float64{1.0, 0.85, 0.7, 0.55, 0.45, 0.3}
	sourceSizeBytes := estimateSourceSizeBytes(info.InputSizeBytes, info.BitrateBps, baseDuration)
	bitrateSteps = chooseBitrateSteps(bitrateSteps, sourceSizeBytes, MaxStickerSizeBytes)
	scaleSteps := []float64{1.0, 0.9, 0.8, 0.7, 0.6}

	loopSeconds := 0
	if kind == InputKindImage || kind == InputKindGIF {
		loopSeconds = DefaultImageDuration
	}

	attempts := make([]EncodeAttempt, 0)
	for _, b := range bitrateSteps {
		attempts = append(attempts, EncodeAttempt{
			Width:           scaled.Width,
			Height:          scaled.Height,
			FPS:             baseAttemptFPS,
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
				FPS:             baseAttemptFPS,
				BitrateKbps:     int(float64(bitrateBase) * b),
				DurationSeconds: baseDuration,
				InputKind:       kind,
				LoopSeconds:     loopSeconds,
			})
		}
	}

	for _, f := range fpsFallbackSteps {
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

func pickBaseAttemptFPS(info MediaInfo, kind InputKind) int {
	if kind == InputKindImage {
		return DefaultImageFPS
	}
	if info.FPS > float64(MaxStickerFPS) {
		return MaxStickerFPS
	}
	return 0
}

func pickFallbackBaseFPS(info MediaInfo, kind InputKind) (int, bool) {
	if kind == InputKindImage {
		return DefaultImageFPS, true
	}
	if info.FPS <= 0 {
		return 0, false
	}
	baseFPS := int(math.Min(info.FPS, float64(MaxStickerFPS)))
	if baseFPS <= 0 {
		return 0, false
	}
	return baseFPS, true
}

func buildFPSFallbackSteps(baseFPS int, allow bool) []int {
	if !allow {
		return nil
	}
	candidates := []int{24, 20, 15}
	steps := make([]int, 0, len(candidates))
	for _, f := range candidates {
		if f > 0 && f < baseFPS {
			steps = append(steps, f)
		}
	}
	return steps
}

func estimateSourceSizeBytes(inputSizeBytes int64, bitrateBps int64, durationSeconds int) int64 {
	sizeByBitrate := int64(0)
	if bitrateBps > 0 && durationSeconds > 0 {
		sizeByBitrate = bitrateBps * int64(durationSeconds) / 8
	}
	if inputSizeBytes > sizeByBitrate {
		return inputSizeBytes
	}
	return sizeByBitrate
}

func chooseBitrateSteps(steps []float64, sourceSizeBytes int64, targetSizeBytes int) []float64 {
	if sourceSizeBytes <= 0 || targetSizeBytes <= 0 {
		return steps
	}
	ratio := float64(targetSizeBytes) / float64(sourceSizeBytes)
	chosen := pickBitrateStep(ratio)
	return reorderSteps(steps, chosen)
}

func pickBitrateStep(ratio float64) float64 {
	if ratio >= 0.9 {
		return 1.0
	}
	if ratio >= 0.7 {
		return 0.85
	}
	if ratio >= 0.5 {
		return 0.7
	}
	return 0.55
}

func reorderSteps(steps []float64, first float64) []float64 {
	reordered := make([]float64, 0, len(steps))
	found := false
	for _, step := range steps {
		if step == first {
			reordered = append(reordered, step)
			found = true
		}
	}
	if !found {
		return steps
	}
	for _, step := range steps {
		if step != first {
			reordered = append(reordered, step)
		}
	}
	return reordered
}
