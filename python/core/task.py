from __future__ import annotations

from dataclasses import dataclass, field
from enum import Enum

from core.job import Job
from core.validate import ValidationIssue


class TaskType(str, Enum):
    VIDEO_STICKER = "video_sticker"
    STATIC_STICKER = "static_sticker"
    EMOJI = "emoji"


@dataclass(frozen=True)
class Task:
    task_id: int
    task_type: TaskType
    label: str
    job: Job


@dataclass
class Result:
    input_path: str
    output_path: str = ""
    err: Exception | None = None
    issues: list[ValidationIssue] = field(default_factory=list)

    def ok(self) -> bool:
        return self.err is None and not self.issues
