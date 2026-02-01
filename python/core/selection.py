from __future__ import annotations

from dataclasses import dataclass, field
from typing import Callable

from core.job import Job, Skipped
from core.media import detect_input_kind
from infra.files import list_files


@dataclass(frozen=True)
class SelectionItem:
    path: str
    is_dir: bool


@dataclass
class ExpandResult:
    jobs: list[Job] = field(default_factory=list)
    dir_count: int = 0
    file_count: int = 0
    total_files: int = 0
    output_dirs: list[str] = field(default_factory=list)
    skipped: list[Skipped] = field(default_factory=list)


class SelectionExpander:
    def __init__(self, list_files_fn: Callable[[str], list[str]] | None = None) -> None:
        self._list_files = list_files_fn or list_files

    def expand(self, selections: list[SelectionItem], output_dir: str) -> ExpandResult:
        result = ExpandResult()
        if not output_dir:
            output_dir = "./output"

        jobs: list[Job] = []
        seen: set[str] = set()
        output_set: set[str] = set()

        files: list[SelectionItem] = []
        dirs: list[SelectionItem] = []
        for selection in selections:
            if selection.is_dir:
                dirs.append(selection)
            else:
                files.append(selection)

        for selection in files:
            try:
                kind = detect_input_kind(selection.path)
            except ValueError as exc:
                result.skipped.append(Skipped(path=selection.path, reason=str(exc)))
                continue
            if selection.path in seen:
                continue
            seen.add(selection.path)
            jobs.append(Job(input_path=selection.path, kind=kind, output_dir=output_dir))
            result.file_count += 1
            result.total_files += 1
            output_set.add(output_dir)

        for selection in dirs:
            files_in_dir = self._list_files(selection.path)
            result.dir_count += 1
            for path in files_in_dir:
                try:
                    kind = detect_input_kind(path)
                except ValueError as exc:
                    result.skipped.append(Skipped(path=path, reason=str(exc)))
                    continue
                if path in seen:
                    continue
                seen.add(path)
                jobs.append(Job(input_path=path, kind=kind, output_dir=output_dir))
                result.total_files += 1
                output_set.add(output_dir)

        result.jobs = jobs
        result.output_dirs = sorted(output_set)
        return result
