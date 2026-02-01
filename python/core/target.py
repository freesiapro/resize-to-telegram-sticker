from __future__ import annotations

from dataclasses import dataclass
from enum import Enum

from core.job import Job
from core.media import InputKind


class TargetType(str, Enum):
    VIDEO_STICKER = "video_sticker"
    STATIC_STICKER = "static_sticker"
    EMOJI = "emoji"


@dataclass
class InputSummary:
    total: int = 0
    image: int = 0
    gif: int = 0
    video: int = 0


class TargetStatus(int, Enum):
    OK = 0
    WARNING = 1
    BLOCKED = 2


@dataclass
class TargetHint:
    status: TargetStatus
    message: str = ""


def target_label(target: TargetType) -> str:
    if target == TargetType.VIDEO_STICKER:
        return "Video Sticker"
    if target == TargetType.STATIC_STICKER:
        return "Static Sticker"
    if target == TargetType.EMOJI:
        return "Emoji"
    return "Unknown"


def summarize_jobs(jobs: list[Job]) -> InputSummary:
    summary = InputSummary()
    for job in jobs:
        summary.total += 1
        if job.kind == InputKind.IMAGE:
            summary.image += 1
        elif job.kind == InputKind.GIF:
            summary.gif += 1
        elif job.kind == InputKind.VIDEO:
            summary.video += 1
    return summary


def evaluate_target(summary: InputSummary, target: TargetType) -> TargetHint:
    if summary.total == 0:
        return TargetHint(status=TargetStatus.BLOCKED, message="No selection")
    allowed = allowed_count(summary, target)
    if allowed == 0:
        return TargetHint(status=TargetStatus.BLOCKED, message=blocked_message(target))
    if allowed < summary.total:
        return TargetHint(status=TargetStatus.WARNING, message=warning_message(target))
    return TargetHint(status=TargetStatus.OK)


def filter_jobs_for_target(jobs: list[Job], target: TargetType) -> list[Job]:
    return [job for job in jobs if is_allowed_kind(job.kind, target)]


def allowed_count(summary: InputSummary, target: TargetType) -> int:
    if target == TargetType.VIDEO_STICKER:
        return summary.video + summary.gif
    if target in (TargetType.STATIC_STICKER, TargetType.EMOJI):
        return summary.image
    return 0


def is_allowed_kind(kind: InputKind, target: TargetType) -> bool:
    if target == TargetType.VIDEO_STICKER:
        return kind in (InputKind.VIDEO, InputKind.GIF)
    if target in (TargetType.STATIC_STICKER, TargetType.EMOJI):
        return kind == InputKind.IMAGE
    return False


def blocked_message(target: TargetType) -> str:
    if target == TargetType.VIDEO_STICKER:
        return "Must select videos or GIFs for this target"
    if target in (TargetType.STATIC_STICKER, TargetType.EMOJI):
        return "Must select images for this target"
    return "No valid inputs"


def warning_message(target: TargetType) -> str:
    if target == TargetType.VIDEO_STICKER:
        return "Only videos or GIFs will be processed"
    if target in (TargetType.STATIC_STICKER, TargetType.EMOJI):
        return "Only images will be processed"
    return "Some inputs will be skipped"
