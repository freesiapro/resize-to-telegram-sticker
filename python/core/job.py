from __future__ import annotations

from dataclasses import dataclass

from core.media import InputKind


@dataclass(frozen=True)
class Job:
    input_path: str
    kind: InputKind
    output_dir: str = ""


@dataclass(frozen=True)
class Skipped:
    path: str
    reason: str
