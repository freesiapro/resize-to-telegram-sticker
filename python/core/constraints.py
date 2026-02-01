from __future__ import annotations

from dataclasses import dataclass

MAX_STICKER_SIDE = 512
MAX_STICKER_FPS = 30
MAX_STICKER_DURATION_SECONDS = 3
MAX_STICKER_SIZE_BYTES = 256 * 1024
DEFAULT_IMAGE_FPS = 30
DEFAULT_IMAGE_DURATION = 3
STATIC_STICKER_SIDE = 512
EMOJI_SIDE = 100


@dataclass(frozen=True)
class Size:
    width: int
    height: int


def scale_to_fit(src: Size, max_side: int) -> Size:
    if src.width <= 0 or src.height <= 0:
        raise ValueError(f"invalid size: {src.width}x{src.height}")

    if src.width == src.height:
        return Size(width=max_side, height=max_side)

    if src.width > src.height:
        height = int(float(src.height) * float(max_side) / float(src.width))
        if height <= 0:
            height = 1
        return Size(width=max_side, height=height)

    width = int(float(src.width) * float(max_side) / float(src.height))
    if width <= 0:
        width = 1
    return Size(width=width, height=max_side)
