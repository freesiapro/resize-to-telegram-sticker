from __future__ import annotations

import os
from dataclasses import dataclass
from pathlib import Path

from typing import Protocol

from core.constraints import MAX_STICKER_DURATION_SECONDS
from core.job import Job
from core.media import MediaInfo
from core.options import EncodeOptions
from core.strategy import EncodeAttempt, build_attempts
from core.task import Result
from core.validate import ValidationIssue, validate_video_output


class ProbeRunner(Protocol):
    def probe(self, media_path: str) -> MediaInfo:  # pragma: no cover - interface
        ...


class EncodeRunner(Protocol):
    def encode(
        self,
        input_path: str,
        attempt: EncodeAttempt,
        output_path: str,
        opts: EncodeOptions,
    ) -> None:  # pragma: no cover - interface
        ...


@dataclass
class VideoPipeline:
    probe: ProbeRunner
    encode: EncodeRunner

    def run(self, jobs: list[Job], cancel_event=None) -> list[Result]:
        results: list[Result] = []
        for job in jobs:
            if cancel_event is not None and cancel_event.is_set():
                results.append(Result(input_path=job.input_path, err=RuntimeError("cancelled")))
                return results

            try:
                info = self.probe.probe(job.input_path)
            except Exception as exc:  # noqa: BLE001
                results.append(Result(input_path=job.input_path, err=exc))
                continue

            try:
                info.input_size_bytes = Path(job.input_path).stat().st_size
            except OSError:
                info.input_size_bytes = 0

            try:
                attempts = build_attempts(info, job.kind)
            except Exception as exc:  # noqa: BLE001
                results.append(Result(input_path=job.input_path, err=exc))
                continue

            if job.output_dir:
                os.makedirs(job.output_dir, exist_ok=True)

            output = output_path(job)
            last_err: Exception | None = None
            last_issues: list[ValidationIssue] = []
            for attempt in attempts:
                if cancel_event is not None and cancel_event.is_set():
                    results.append(Result(input_path=job.input_path, err=RuntimeError("cancelled")))
                    return results
                try:
                    self.encode.encode(
                        job.input_path,
                        attempt,
                        output,
                        EncodeOptions(trim_seconds=MAX_STICKER_DURATION_SECONDS),
                    )
                except Exception as exc:  # noqa: BLE001
                    last_err = exc
                    continue

                try:
                    size_bytes = Path(output).stat().st_size
                except OSError as exc:
                    last_err = exc
                    continue

                try:
                    out_info = self.probe.probe(output)
                except Exception as exc:  # noqa: BLE001
                    last_err = exc
                    continue

                issues = validate_video_output(out_info, size_bytes)
                if not issues:
                    results.append(Result(input_path=job.input_path, output_path=output))
                    last_err = None
                    break
                last_issues = issues
                last_err = RuntimeError("validation failed")

            if last_err is not None:
                results.append(
                    Result(input_path=job.input_path, err=last_err, issues=last_issues)
                )

        return results


def output_path(job: Job) -> str:
    base_name = Path(job.input_path).stem
    name = f"{base_name}_sticker.webm"
    if not job.output_dir:
        return str(Path(job.input_path).with_name(name))
    return str(Path(job.output_dir) / name)
