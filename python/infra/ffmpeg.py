from __future__ import annotations

from dataclasses import dataclass

import ffmpeg

from core.media import InputKind
from core.options import EncodeOptions, ImageEncodeOptions
from core.strategy import EncodeAttempt


@dataclass
class FFmpegRunner:
    def encode(
        self,
        input_path: str,
        attempt: EncodeAttempt,
        output_path: str,
        opts: EncodeOptions,
    ) -> None:
        input_kwargs = build_input_kwargs(attempt)
        stream = ffmpeg.input(input_path, **input_kwargs)

        stream = stream.filter("scale", attempt.width, attempt.height)

        if attempt.fps > 0:
            stream = stream.filter("fps", attempt.fps)

        if opts.trim_seconds > 0:
            stream = stream.trim(duration=opts.trim_seconds).setpts("PTS-STARTPTS")

        output_kwargs = build_output_kwargs(attempt)
        try:
            (
                ffmpeg.output(stream, output_path, **output_kwargs)
                .overwrite_output()
                .run(capture_stdout=True, capture_stderr=True)
            )
        except ffmpeg.Error as exc:
            stdout_text = decode_stream(exc.stdout)
            stderr_text = decode_stream(exc.stderr)
            log_path, log_err = write_ffmpeg_error_log(output_path, stdout_text, stderr_text)
            suffix = format_ffmpeg_stderr(stderr_text)
            if log_err is None and log_path:
                suffix = f"{suffix} (ffmpeg log: {log_path})"
            elif log_err is not None:
                suffix = f"{suffix} (ffmpeg log write failed: {log_err})"
            raise RuntimeError(f"ffmpeg failed: {exc}{suffix}") from exc

    def encode_image(
        self, input_path: str, opts: ImageEncodeOptions, output_path: str
    ) -> None:
        stream = ffmpeg.input(input_path)

        stream = stream.filter(
            "scale",
            opts.target_side,
            opts.target_side,
            force_original_aspect_ratio="decrease",
        )
        if opts.pad_to_square:
            stream = stream.filter(
                "pad",
                opts.target_side,
                opts.target_side,
                "(ow-iw)/2",
                "(oh-ih)/2",
                color="0x00000000",
            )

        output_kwargs = build_image_output_kwargs()
        try:
            (
                ffmpeg.output(stream, output_path, **output_kwargs)
                .overwrite_output()
                .run(capture_stdout=True, capture_stderr=True)
            )
        except ffmpeg.Error as exc:
            stdout_text = decode_stream(exc.stdout)
            stderr_text = decode_stream(exc.stderr)
            log_path, log_err = write_ffmpeg_error_log(output_path, stdout_text, stderr_text)
            suffix = format_ffmpeg_stderr(stderr_text)
            if log_err is None and log_path:
                suffix = f"{suffix} (ffmpeg log: {log_path})"
            elif log_err is not None:
                suffix = f"{suffix} (ffmpeg log write failed: {log_err})"
            raise RuntimeError(f"ffmpeg failed: {exc}{suffix}") from exc


def build_input_kwargs(attempt: EncodeAttempt) -> dict:
    kwargs: dict = {}
    if attempt.input_kind == InputKind.IMAGE:
        kwargs["loop"] = 1
    if attempt.input_kind == InputKind.GIF:
        kwargs["stream_loop"] = -1
    return kwargs


def build_output_kwargs(attempt: EncodeAttempt) -> dict:
    kwargs: dict = {"c:v": "libvpx-vp9", "an": None}
    if attempt.bitrate_kbps > 0:
        kwargs["b:v"] = f"{attempt.bitrate_kbps}k"
    if attempt.fps > 0:
        kwargs["r"] = str(attempt.fps)
    else:
        kwargs["fps_mode"] = "vfr"
    if attempt.duration_seconds > 0:
        kwargs["t"] = str(attempt.duration_seconds)
    return kwargs


def build_image_output_kwargs() -> dict:
    return {"vframes": 1, "vcodec": "png", "f": "image2"}


def format_ffmpeg_stderr(stderr: str) -> str:
    trimmed = stderr.strip()
    if not trimmed:
        return ""
    max_len = 2048
    if len(trimmed) > max_len:
        trimmed = trimmed[-max_len:]
    return f": {trimmed}"


def write_ffmpeg_error_log(
    output_path: str, stdout: str, stderr: str
) -> tuple[str, Exception | None]:
    if not output_path:
        return "", ValueError("empty output path")
    log_path = f"{output_path}.ffmpeg-error.log"
    content = format_ffmpeg_log(stdout, stderr)
    try:
        with open(log_path, "w", encoding="utf-8") as handle:
            handle.write(content)
    except Exception as exc:  # noqa: BLE001 - keep parity with Go behavior
        return "", exc
    return log_path, None


def format_ffmpeg_log(stdout: str, stderr: str) -> str:
    def write_section(title: str, content: str) -> str:
        if not content.strip():
            return f"{title}:\n<empty>\n"
        if content.endswith("\n"):
            return f"{title}:\n{content}"
        return f"{title}:\n{content}\n"

    return write_section("STDOUT", stdout) + write_section("STDERR", stderr)


def decode_stream(value: bytes | None) -> str:
    if value is None:
        return ""
    try:
        return value.decode("utf-8", errors="replace")
    except AttributeError:
        return str(value)
