from __future__ import annotations

from dataclasses import dataclass

from core.constraints import (
    EMOJI_SIDE,
    MAX_STICKER_DURATION_SECONDS,
    MAX_STICKER_FPS,
    MAX_STICKER_SIDE,
    MAX_STICKER_SIZE_BYTES,
    STATIC_STICKER_SIDE,
)
from core.media import MediaInfo


@dataclass(frozen=True)
class ValidationIssue:
    code: str
    message: str


@dataclass(frozen=True)
class ImageInfo:
    width: int
    height: int
    format: str


def validate_video_output(info: MediaInfo, size_bytes: int) -> list[ValidationIssue]:
    issues: list[ValidationIssue] = []
    if size_bytes > MAX_STICKER_SIZE_BYTES:
        issues.append(ValidationIssue(code="size", message="size exceeds limit"))
    if info.fps > MAX_STICKER_FPS:
        issues.append(ValidationIssue(code="fps", message="fps exceeds limit"))
    if info.duration_seconds > float(MAX_STICKER_DURATION_SECONDS):
        issues.append(ValidationIssue(code="duration", message="duration exceeds limit"))
    if info.has_audio:
        issues.append(ValidationIssue(code="audio", message="audio stream present"))
    if "vp9" not in info.codec_name.lower():
        issues.append(ValidationIssue(code="codec", message="codec is not vp9"))
    if "webm" not in info.format_name.lower():
        issues.append(ValidationIssue(code="format", message="format is not webm"))
    if info.width != MAX_STICKER_SIDE and info.height != MAX_STICKER_SIDE:
        issues.append(ValidationIssue(code="size", message="one side must be 512"))
    if info.width > MAX_STICKER_SIDE or info.height > MAX_STICKER_SIDE:
        issues.append(ValidationIssue(code="size", message="dimension exceeds 512"))
    return issues


def validate_static_sticker_image(info: ImageInfo) -> list[ValidationIssue]:
    issues: list[ValidationIssue] = []
    if not is_png(info.format):
        issues.append(ValidationIssue(code="format", message="format is not png"))
    if info.width != STATIC_STICKER_SIDE and info.height != STATIC_STICKER_SIDE:
        issues.append(ValidationIssue(code="size", message="one side must be 512"))
    if info.width > STATIC_STICKER_SIDE or info.height > STATIC_STICKER_SIDE:
        issues.append(ValidationIssue(code="size", message="dimension exceeds 512"))
    return issues


def validate_emoji_image(info: ImageInfo) -> list[ValidationIssue]:
    issues: list[ValidationIssue] = []
    if not is_png(info.format):
        issues.append(ValidationIssue(code="format", message="format is not png"))
    if info.width != EMOJI_SIDE or info.height != EMOJI_SIDE:
        issues.append(ValidationIssue(code="size", message="dimension must be 100x100"))
    return issues


def is_png(fmt: str) -> bool:
    return fmt.lower() == "png"
