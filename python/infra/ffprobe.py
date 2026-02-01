from __future__ import annotations

import json
import subprocess
from dataclasses import dataclass

from core.media import MediaInfo


@dataclass
class FFprobeRunner:
    path: str = "ffprobe"

    def probe(self, media_path: str) -> MediaInfo:
        args = [
            self.path,
            "-v",
            "error",
            "-show_entries",
            "stream=codec_type,width,height,r_frame_rate,codec_name,duration",
            "-show_entries",
            "format=duration,format_name,bit_rate",
            "-of",
            "json",
            media_path,
        ]
        proc = subprocess.run(args, capture_output=True, text=True, check=False)
        if proc.returncode != 0:
            stderr = proc.stderr.strip()
            raise RuntimeError(f"ffprobe failed: {stderr}")
        return parse_probe_json(proc.stdout)


def parse_probe_json(payload: str) -> MediaInfo:
    data = json.loads(payload)
    info = MediaInfo()

    for stream in data.get("streams", []):
        codec_type = stream.get("codec_type")
        if codec_type == "audio":
            info.has_audio = True
        if codec_type == "video":
            info.width = int(stream.get("width") or 0)
            info.height = int(stream.get("height") or 0)
            info.fps = parse_frame_rate(str(stream.get("r_frame_rate") or "0/0"))
            info.codec_name = str(stream.get("codec_name") or "")
            if info.duration_seconds == 0:
                info.duration_seconds = parse_duration(str(stream.get("duration") or "0"))

    fmt = data.get("format", {})
    if info.duration_seconds == 0:
        info.duration_seconds = parse_duration(str(fmt.get("duration") or "0"))
    info.format_name = str(fmt.get("format_name") or "")
    info.bitrate_bps = parse_bitrate(str(fmt.get("bit_rate") or "0"))

    return info


def parse_frame_rate(value: str) -> float:
    parts = value.split("/")
    if len(parts) != 2:
        return 0.0
    try:
        num = float(parts[0])
        den = float(parts[1])
    except ValueError:
        return 0.0
    if den == 0:
        return 0.0
    return num / den


def parse_duration(value: str) -> float:
    try:
        return float(value)
    except ValueError:
        return 0.0


def parse_bitrate(value: str) -> int:
    if not value:
        return 0
    try:
        return int(value)
    except ValueError:
        try:
            return int(float(value))
        except ValueError:
            return 0
