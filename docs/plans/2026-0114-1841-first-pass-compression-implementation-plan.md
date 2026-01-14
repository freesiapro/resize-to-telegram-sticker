# First-pass Compression Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Use input file size + source bitrate to choose an adaptive first-pass bitrate so outputs more often meet the 256KB sticker limit in one pass.

**Architecture:** Extend `MediaInfo` to carry input size and bitrate; parse `bit_rate` in ffprobe; compute `sourceSize` and reorder bitrate steps in `BuildAttempts`; set `InputSizeBytes` in pipeline after probe. All other attempts/validation stay unchanged.

**Tech Stack:** Go (std lib + testing), ffprobe CLI output parsing.

### Task 1: Parse bitrate from ffprobe and expose in MediaInfo

**Files:**
- Modify: `internal/domain/media.go`
- Modify: `internal/infra/ffprobe.go`
- Test: `internal/infra/ffprobe_test.go`

**Step 1: Write the failing test**

Update `TestParseProbe` to include `bit_rate` and assert it is parsed:

```go
jsonStr := `{"streams":[{"codec_type":"video","width":512,"height":256,"r_frame_rate":"30/1","codec_name":"vp9"},{"codec_type":"audio"}],"format":{"format_name":"webm","duration":"2.9","bit_rate":"1234567"}}`
info, err := parseProbeJSON([]byte(jsonStr))
if err != nil {
	t.Fatalf("unexpected err: %v", err)
}
if info.BitrateBps != 1234567 {
	t.Fatalf("unexpected bitrate: %d", info.BitrateBps)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/infra -run TestParseProbe`  
Expected: FAIL (missing field or zero bitrate).

**Step 3: Write minimal implementation**

Add fields and parsing:

```go
// internal/domain/media.go
type MediaInfo struct {
	// ...
	BitrateBps     int64
	InputSizeBytes int64
}

// internal/infra/ffprobe.go
type probeJSON struct {
	// ...
	Format struct {
		FormatName string `json:"format_name"`
		Duration   string `json:"duration"`
		BitRate    string `json:"bit_rate"`
	} `json:"format"`
}

func parseProbeJSON(data []byte) (domain.MediaInfo, error) {
	// ...
	info.FormatName = p.Format.FormatName
	info.BitrateBps = parseBitRate(p.Format.BitRate)
	return info, nil
}

func parseBitRate(v string) int64 {
	if v == "" {
		return 0
	}
	rate, err := strconv.ParseInt(v, 10, 64)
	if err == nil {
		return rate
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0
	}
	return int64(f)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/infra -run TestParseProbe`  
Expected: PASS.

### Task 2: Adaptive first-pass bitrate selection in BuildAttempts

**Files:**
- Modify: `internal/domain/strategy.go`
- Test: `internal/domain/strategy_test.go`

**Step 1: Write the failing test**

Add a new table-driven test to assert the chosen first-pass multiplier:

```go
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/domain -run TestBuildAttemptsAdaptiveBitrateStep`  
Expected: FAIL (logic not implemented).

**Step 3: Write minimal implementation**

Update bitrate selection and reorder steps:

```go
func BuildAttempts(info MediaInfo, kind InputKind) ([]EncodeAttempt, error) {
	// existing setup...
	sourceSizeBytes := estimateSourceSizeBytes(info.InputSizeBytes, info.BitrateBps, baseDuration)
	bitrateSteps := chooseBitrateSteps(
		[]float64{1.0, 0.85, 0.7, 0.55},
		sourceSizeBytes,
		MaxStickerSizeBytes,
	)
	// rest unchanged...
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
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/domain -run TestBuildAttemptsAdaptiveBitrateStep`  
Expected: PASS.

### Task 3: Wire input file size into MediaInfo in pipeline + integration test

**Files:**
- Modify: `internal/app/pipeline.go`
- Test: `internal/app/pipeline_test.go`

**Step 1: Write the failing test**

Add a test that captures the first attempt bitrate using a large temp file:

```go
type captureFirstAttempt struct {
	first domain.EncodeAttempt
	seen  bool
}

func (c *captureFirstAttempt) Encode(_ context.Context, _ string, attempt domain.EncodeAttempt, _ string, _ domain.EncodeOptions) error {
	if !c.seen {
		c.first = attempt
		c.seen = true
	}
	return errors.New("encode failed")
}

func TestPipelineUsesInputSizeForFirstAttempt(t *testing.T) {
	tempFile, err := os.CreateTemp(t.TempDir(), "input-*.mp4")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	payload := make([]byte, 1024*1024)
	if _, err := tempFile.Write(payload); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	encoder := &captureFirstAttempt{}
	p := Pipeline{
		Probe: fakeProbe{info: domain.MediaInfo{Width: 512, Height: 256, FPS: 30, DurationSeconds: 3}},
		Encode: encoder,
	}
	_ = p.Run(context.Background(), []Job{{InputPath: tempFile.Name(), Kind: domain.InputKindVideo}})

	baseBitrate := expectedBaseBitrateKbps(3)
	want := int(float64(baseBitrate) * 0.55)
	if encoder.first.BitrateKbps != want {
		t.Fatalf("unexpected first bitrate: %d want %d", encoder.first.BitrateKbps, want)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/app -run TestPipelineUsesInputSizeForFirstAttempt`  
Expected: FAIL (first attempt still uses 1.0 step).

**Step 3: Write minimal implementation**

Set `InputSizeBytes` from `os.Stat` when available:

```go
info, err := p.Probe.Probe(ctx, job.InputPath)
if err != nil {
	// existing error handling
}
if stat, statErr := os.Stat(job.InputPath); statErr == nil {
	info.InputSizeBytes = stat.Size()
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/app -run TestPipelineUsesInputSizeForFirstAttempt`  
Expected: PASS.

### Task 4: Final verification (exclude UI tests)

Run: `go test ./internal/domain ./internal/infra ./internal/app`  
Expected: PASS (UI tests skipped per request).

