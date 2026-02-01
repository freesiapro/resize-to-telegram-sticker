from __future__ import annotations

from dataclasses import dataclass

from PySide6.QtCore import QObject, QRunnable, Signal

from core.job import Job
from core.pipeline.image import ImagePipeline
from core.pipeline.video import VideoPipeline
from core.target import TargetType
from core.task import Result


@dataclass(frozen=True)
class WorkerContext:
    job: Job
    target_type: TargetType
    video_pipeline: VideoPipeline
    image_pipeline: ImagePipeline
    cancel_event: object | None


class WorkerSignals(QObject):
    started = Signal(int, str)
    finished = Signal(int, object)


class TaskWorker(QRunnable):
    def __init__(self, index: int, context: WorkerContext) -> None:
        super().__init__()
        self.index = index
        self.context = context
        self.signals = WorkerSignals()

    def run(self) -> None:
        job = self.context.job
        self.signals.started.emit(self.index, job.input_path)
        result = process_job(self.context)
        self.signals.finished.emit(self.index, result)


def process_job(context: WorkerContext) -> Result:
    job = context.job
    cancel_event = context.cancel_event
    if cancel_event is not None and getattr(cancel_event, "is_set", lambda: False)():
        return Result(input_path=job.input_path, err=RuntimeError("cancelled"))

    try:
        if context.target_type == TargetType.VIDEO_STICKER:
            results = context.video_pipeline.run([job], cancel_event=cancel_event)
        else:
            results = context.image_pipeline.run(
                [job], target_type=context.target_type, cancel_event=cancel_event
            )
    except Exception as exc:  # noqa: BLE001
        return Result(input_path=job.input_path, err=exc)

    if not results:
        return Result(input_path=job.input_path, err=RuntimeError("no result"))
    return results[0]
