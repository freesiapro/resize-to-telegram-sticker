from __future__ import annotations

import math
from dataclasses import dataclass

from core.constraints import (
    DEFAULT_IMAGE_DURATION,
    DEFAULT_IMAGE_FPS,
    MAX_STICKER_DURATION_SECONDS,
    MAX_STICKER_FPS,
    MAX_STICKER_SIDE,
    MAX_STICKER_SIZE_BYTES,
    Size,
    scale_to_fit,
)
from core.media import InputKind, MediaInfo


@dataclass(frozen=True)
class EncodeAttempt:
    width: int
    height: int
    fps: int
    bitrate_kbps: int
    duration_seconds: int
    input_kind: InputKind
    loop_seconds: int


def build_attempts(info: MediaInfo, kind: InputKind) -> list[EncodeAttempt]:
    scaled = scale_to_fit(Size(width=info.width, height=info.height), MAX_STICKER_SIDE)

    base_attempt_fps = pick_base_attempt_fps(info, kind)
    fallback_base_fps, allow_fps_fallback = pick_fallback_base_fps(info, kind)
    fps_fallback_steps = build_fps_fallback_steps(fallback_base_fps, allow_fps_fallback)

    base_duration = MAX_STICKER_DURATION_SECONDS
    if 0 < info.duration_seconds < float(MAX_STICKER_DURATION_SECONDS):
        base_duration = int(math.ceil(info.duration_seconds))

    if kind in (InputKind.IMAGE, InputKind.GIF):
        base_duration = DEFAULT_IMAGE_DURATION

    if base_duration <= 0:
        base_duration = MAX_STICKER_DURATION_SECONDS

    bitrate_base = int(float(MAX_STICKER_SIZE_BYTES * 8) / float(base_duration) / 1000.0)
    if bitrate_base < 150:
        bitrate_base = 150

    bitrate_steps = [1.0, 0.85, 0.7, 0.55, 0.45, 0.3]
    source_size_bytes = estimate_source_size_bytes(
        info.input_size_bytes, info.bitrate_bps, base_duration
    )
    bitrate_steps = choose_bitrate_steps(bitrate_steps, source_size_bytes, MAX_STICKER_SIZE_BYTES)

    scale_steps = [1.0, 0.9, 0.8, 0.7, 0.6]

    loop_seconds = DEFAULT_IMAGE_DURATION if kind in (InputKind.IMAGE, InputKind.GIF) else 0

    attempts: list[EncodeAttempt] = []
    for step in bitrate_steps:
        attempts.append(
            EncodeAttempt(
                width=scaled.width,
                height=scaled.height,
                fps=base_attempt_fps,
                bitrate_kbps=int(float(bitrate_base) * step),
                duration_seconds=base_duration,
                input_kind=kind,
                loop_seconds=loop_seconds,
            )
        )

    for scale in scale_steps[1:]:
        width = int(float(scaled.width) * scale)
        height = int(float(scaled.height) * scale)
        if width <= 0:
            width = 1
        if height <= 0:
            height = 1
        for step in bitrate_steps:
            attempts.append(
                EncodeAttempt(
                    width=width,
                    height=height,
                    fps=base_attempt_fps,
                    bitrate_kbps=int(float(bitrate_base) * step),
                    duration_seconds=base_duration,
                    input_kind=kind,
                    loop_seconds=loop_seconds,
                )
            )

    for fps in fps_fallback_steps:
        for step in bitrate_steps:
            attempts.append(
                EncodeAttempt(
                    width=scaled.width,
                    height=scaled.height,
                    fps=fps,
                    bitrate_kbps=int(float(bitrate_base) * step),
                    duration_seconds=base_duration,
                    input_kind=kind,
                    loop_seconds=loop_seconds,
                )
            )

    return attempts


def pick_base_attempt_fps(info: MediaInfo, kind: InputKind) -> int:
    if kind == InputKind.IMAGE:
        return DEFAULT_IMAGE_FPS
    if info.fps > float(MAX_STICKER_FPS):
        return MAX_STICKER_FPS
    return 0


def pick_fallback_base_fps(info: MediaInfo, kind: InputKind) -> tuple[int, bool]:
    if kind == InputKind.IMAGE:
        return DEFAULT_IMAGE_FPS, True
    if info.fps <= 0:
        return 0, False
    base_fps = int(min(info.fps, float(MAX_STICKER_FPS)))
    if base_fps <= 0:
        return 0, False
    return base_fps, True


def build_fps_fallback_steps(base_fps: int, allow: bool) -> list[int]:
    if not allow:
        return []
    candidates = [24, 20, 15]
    steps: list[int] = []
    for fps in candidates:
        if fps > 0 and fps < base_fps:
            steps.append(fps)
    return steps


def estimate_source_size_bytes(
    input_size_bytes: int, bitrate_bps: int, duration_seconds: int
) -> int:
    size_by_bitrate = 0
    if bitrate_bps > 0 and duration_seconds > 0:
        size_by_bitrate = int(bitrate_bps) * int(duration_seconds) // 8
    return max(int(input_size_bytes), int(size_by_bitrate))


def choose_bitrate_steps(
    steps: list[float], source_size_bytes: int, target_size_bytes: int
) -> list[float]:
    if source_size_bytes <= 0 or target_size_bytes <= 0:
        return steps
    ratio = float(target_size_bytes) / float(source_size_bytes)
    chosen = pick_bitrate_step(ratio)
    return reorder_steps(steps, chosen)


def pick_bitrate_step(ratio: float) -> float:
    if ratio >= 0.9:
        return 1.0
    if ratio >= 0.7:
        return 0.85
    if ratio >= 0.5:
        return 0.7
    return 0.55


def reorder_steps(steps: list[float], first: float) -> list[float]:
    reordered: list[float] = []
    found = False
    for step in steps:
        if step == first:
            reordered.append(step)
            found = True
    if not found:
        return steps
    for step in steps:
        if step != first:
            reordered.append(step)
    return reordered
