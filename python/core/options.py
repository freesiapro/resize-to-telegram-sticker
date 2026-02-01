from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class EncodeOptions:
    trim_seconds: int = 0


@dataclass(frozen=True)
class ImageEncodeOptions:
    target_side: int
    pad_to_square: bool = False
