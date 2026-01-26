package domain

import (
	"fmt"
	"path/filepath"
	"strings"
)

type InputKind string

const (
	InputKindVideo InputKind = "video"
	InputKindImage InputKind = "image"
	InputKindGIF   InputKind = "gif"
)

type MediaInfo struct {
	Width           int
	Height          int
	FPS             float64
	DurationSeconds float64
	HasAudio        bool
	FormatName      string
	CodecName       string
	BitrateBps      int64
	InputSizeBytes  int64
}

var (
	videoExts = map[string]struct{}{".mp4": {}, ".mov": {}, ".webm": {}, ".mkv": {}, ".avi": {}}
	imageExts = map[string]struct{}{".png": {}, ".jpg": {}, ".jpeg": {}, ".webp": {}}
	gifExts   = map[string]struct{}{".gif": {}}
)

func DetectInputKind(path string) (InputKind, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if _, ok := gifExts[ext]; ok {
		return InputKindGIF, nil
	}
	if _, ok := imageExts[ext]; ok {
		return InputKindImage, nil
	}
	if _, ok := videoExts[ext]; ok {
		return InputKindVideo, nil
	}
	return "", fmt.Errorf("unsupported input: %s", path)
}
