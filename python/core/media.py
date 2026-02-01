from __future__ import annotations

from dataclasses import dataclass
from enum import Enum
from pathlib import Path


class InputKind(str, Enum):
    VIDEO = "video"
    IMAGE = "image"
    GIF = "gif"


@dataclass
class MediaInfo:
    width: int = 0
    height: int = 0
    fps: float = 0.0
    duration_seconds: float = 0.0
    has_audio: bool = False
    format_name: str = ""
    codec_name: str = ""
    bitrate_bps: int = 0
    input_size_bytes: int = 0


VIDEO_EXTS = {".mp4", ".mov", ".webm", ".mkv", ".avi"}
IMAGE_EXTS = {".png", ".jpg", ".jpeg", ".webp"}
GIF_EXTS = {".gif"}


def detect_input_kind(path: str) -> InputKind:
    ext = Path(path).suffix.lower()
    if ext in GIF_EXTS:
        return InputKind.GIF
    if ext in IMAGE_EXTS:
        return InputKind.IMAGE
    if ext in VIDEO_EXTS:
        return InputKind.VIDEO
    raise ValueError(f"unsupported input: {path}")
