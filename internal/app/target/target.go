package target

import (
	"github.com/freesiapro/resize-to-telegram-sticker/internal/app/job"
	"github.com/freesiapro/resize-to-telegram-sticker/internal/domain"
)

type TargetType string

const (
	TargetVideoSticker  TargetType = "video_sticker"
	TargetStaticSticker TargetType = "static_sticker"
	TargetEmoji         TargetType = "emoji"
)

type InputSummary struct {
	Total int
	Image int
	GIF   int
	Video int
}

type TargetStatus int

const (
	TargetStatusOK TargetStatus = iota
	TargetStatusWarning
	TargetStatusBlocked
)

type TargetHint struct {
	Status  TargetStatus
	Message string
}

func TargetLabel(target TargetType) string {
	switch target {
	case TargetVideoSticker:
		return "Video Sticker"
	case TargetStaticSticker:
		return "Static Sticker"
	case TargetEmoji:
		return "Emoji"
	default:
		return "Unknown"
	}
}

func SummarizeJobs(jobs []job.Job) InputSummary {
	summary := InputSummary{}
	for _, job := range jobs {
		summary.Total++
		switch job.Kind {
		case domain.InputKindImage:
			summary.Image++
		case domain.InputKindGIF:
			summary.GIF++
		case domain.InputKindVideo:
			summary.Video++
		}
	}
	return summary
}

func EvaluateTarget(summary InputSummary, target TargetType) TargetHint {
	if summary.Total == 0 {
		return TargetHint{Status: TargetStatusBlocked, Message: "No selection"}
	}
	allowed := allowedCount(summary, target)
	if allowed == 0 {
		return TargetHint{Status: TargetStatusBlocked, Message: blockedMessage(target)}
	}
	if allowed < summary.Total {
		return TargetHint{Status: TargetStatusWarning, Message: warningMessage(target)}
	}
	return TargetHint{Status: TargetStatusOK}
}

func FilterJobsForTarget(jobs []job.Job, target TargetType) []job.Job {
	filtered := make([]job.Job, 0, len(jobs))
	for _, job := range jobs {
		if isAllowedKind(job.Kind, target) {
			filtered = append(filtered, job)
		}
	}
	return filtered
}

func allowedCount(summary InputSummary, target TargetType) int {
	switch target {
	case TargetVideoSticker:
		return summary.Video + summary.GIF
	case TargetStaticSticker, TargetEmoji:
		return summary.Image
	default:
		return 0
	}
}

func isAllowedKind(kind domain.InputKind, target TargetType) bool {
	switch target {
	case TargetVideoSticker:
		return kind == domain.InputKindVideo || kind == domain.InputKindGIF
	case TargetStaticSticker, TargetEmoji:
		return kind == domain.InputKindImage
	default:
		return false
	}
}

func blockedMessage(target TargetType) string {
	switch target {
	case TargetVideoSticker:
		return "Must select videos or GIFs for this target"
	case TargetStaticSticker, TargetEmoji:
		return "Must select images for this target"
	default:
		return "No valid inputs"
	}
}

func warningMessage(target TargetType) string {
	switch target {
	case TargetVideoSticker:
		return "Only videos or GIFs will be processed"
	case TargetStaticSticker, TargetEmoji:
		return "Only images will be processed"
	default:
		return "Some inputs will be skipped"
	}
}
