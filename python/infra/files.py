from __future__ import annotations

import os


def list_files(root: str) -> list[str]:
    files: list[str] = []
    for dirpath, _, filenames in os.walk(root):
        for name in filenames:
            files.append(os.path.join(dirpath, name))
    return files
