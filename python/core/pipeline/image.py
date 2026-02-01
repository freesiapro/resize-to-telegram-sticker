from __future__ import annotations

import os
from dataclasses import dataclass
from typing import Protocol
from pathlib import Path

from PIL import Image

from core.constraints import EMOJI_SIDE, STATIC_STICKER_SIDE
from core.job import Job
from core.media import InputKind
from core.options import ImageEncodeOptions
from core.target import TargetType
from core.task import Result
from core.validate import ImageInfo, ValidationIssue, validate_emoji_image, validate_static_sticker_image


class ImageEncodeRunner(Protocol):
    def encode_image(
        self, input_path: str, opts: ImageEncodeOptions, output_path: str
    ) -> None:  # pragma: no cover - interface
        ...


@dataclass
class ImagePipeline:
    encode: ImageEncodeRunner

    def run(self, jobs: list[Job], target_type: TargetType, cancel_event=None) -> list[Result]:
        results: list[Result] = []
        for job in jobs:
            if cancel_event is not None and cancel_event.is_set():
                results.append(Result(input_path=job.input_path, err=RuntimeError("cancelled")))
                return results

            if job.kind != InputKind.IMAGE:
                results.append(
                    Result(
                        input_path=job.input_path,
                        err=RuntimeError("unsupported input kind"),
                    )
                )
                continue

            if job.output_dir:
                os.makedirs(job.output_dir, exist_ok=True)

            output = image_output_path(job, target_type)
            try:
                opts = image_encode_options(target_type)
            except Exception as exc:  # noqa: BLE001
                results.append(Result(input_path=job.input_path, err=exc))
                continue

            try:
                self.encode.encode_image(job.input_path, opts, output)
            except Exception as exc:  # noqa: BLE001
                results.append(Result(input_path=job.input_path, err=exc))
                continue

            try:
                info = probe_image_info(output)
            except Exception as exc:  # noqa: BLE001
                results.append(Result(input_path=job.input_path, err=exc))
                continue

            issues = validate_image_output(info, target_type)
            if not issues:
                results.append(Result(input_path=job.input_path, output_path=output))
                continue
            results.append(
                Result(
                    input_path=job.input_path,
                    err=RuntimeError("validation failed"),
                    issues=issues,
                )
            )
        return results


def image_output_path(job: Job, target_type: TargetType) -> str:
    base_name = Path(job.input_path).stem
    suffix = "_sticker"
    if target_type == TargetType.EMOJI:
        suffix = "_emoji"
    name = f"{base_name}{suffix}.png"
    if not job.output_dir:
        return str(Path(job.input_path).with_name(name))
    return str(Path(job.output_dir) / name)


def image_encode_options(target_type: TargetType) -> ImageEncodeOptions:
    if target_type == TargetType.STATIC_STICKER:
        return ImageEncodeOptions(target_side=STATIC_STICKER_SIDE, pad_to_square=False)
    if target_type == TargetType.EMOJI:
        return ImageEncodeOptions(target_side=EMOJI_SIDE, pad_to_square=True)
    raise ValueError("unsupported target")


def validate_image_output(info: ImageInfo, target_type: TargetType) -> list[ValidationIssue]:
    if target_type == TargetType.STATIC_STICKER:
        return validate_static_sticker_image(info)
    if target_type == TargetType.EMOJI:
        return validate_emoji_image(info)
    return [ValidationIssue(code="target", message="unsupported target")]


def probe_image_info(path: str) -> ImageInfo:
    with Image.open(path) as img:
        width, height = img.size
        fmt = img.format or ""
    return ImageInfo(width=width, height=height, format=fmt.lower())
