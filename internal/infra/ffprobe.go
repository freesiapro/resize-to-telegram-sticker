package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/freesiapro/resize-to-telegram-sticker/internal/domain"
)

type FFprobeRunner struct {
	Path string
}

type probeJSON struct {
	Streams []struct {
		CodecType string `json:"codec_type"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
		FrameRate string `json:"r_frame_rate"`
		CodecName string `json:"codec_name"`
		Duration  string `json:"duration"`
	} `json:"streams"`
	Format struct {
		FormatName string `json:"format_name"`
		Duration   string `json:"duration"`
		BitRate    string `json:"bit_rate"`
	} `json:"format"`
}

func (r FFprobeRunner) Probe(ctx context.Context, path string) (domain.MediaInfo, error) {
	bin := r.Path
	if bin == "" {
		bin = "ffprobe"
	}
	args := []string{
		"-v", "error",
		"-show_entries", "stream=codec_type,width,height,r_frame_rate,codec_name,duration",
		"-show_entries", "format=duration,format_name",
		"-of", "json",
		path,
	}
	out, err := exec.CommandContext(ctx, bin, args...).Output()
	if err != nil {
		return domain.MediaInfo{}, fmt.Errorf("ffprobe failed: %w", err)
	}
	return parseProbeJSON(out)
}

func parseProbeJSON(data []byte) (domain.MediaInfo, error) {
	var p probeJSON
	if err := json.Unmarshal(data, &p); err != nil {
		return domain.MediaInfo{}, err
	}

	info := domain.MediaInfo{}
	for _, s := range p.Streams {
		if s.CodecType == "audio" {
			info.HasAudio = true
		}
		if s.CodecType == "video" {
			info.Width = s.Width
			info.Height = s.Height
			info.FPS = parseFrameRate(s.FrameRate)
			info.CodecName = s.CodecName
			if info.DurationSeconds == 0 {
				info.DurationSeconds = parseDuration(s.Duration)
			}
		}
	}
	if info.DurationSeconds == 0 {
		info.DurationSeconds = parseDuration(p.Format.Duration)
	}
	info.FormatName = p.Format.FormatName
	info.BitrateBps = parseBitRate(p.Format.BitRate)

	return info, nil
}

func parseFrameRate(v string) float64 {
	parts := strings.Split(v, "/")
	if len(parts) != 2 {
		return 0
	}
	num, _ := strconv.ParseFloat(parts[0], 64)
	den, _ := strconv.ParseFloat(parts[1], 64)
	if den == 0 {
		return 0
	}
	return num / den
}

func parseDuration(v string) float64 {
	f, _ := strconv.ParseFloat(v, 64)
	return f
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
