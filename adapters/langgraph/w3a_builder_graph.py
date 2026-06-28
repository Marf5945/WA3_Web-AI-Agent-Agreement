"""Minimal LangGraph-style wrapper for the W3A builder CLI.

This file is an integration sketch, not a required runtime dependency. Keep the
security decision boundary in the W3A CLI or a deterministic equivalent.
"""

from __future__ import annotations

import json
import subprocess
from pathlib import Path
from typing import Any


def run_w3a_build(package_root: str, answers_path: str, out_path: str) -> dict[str, Any]:
    root = Path(package_root)
    cmd = [
        "go",
        "run",
        "./tools/w3a",
        "build",
        "--answers",
        str(Path(answers_path)),
        "--out",
        str(Path(out_path)),
    ]
    proc = subprocess.run(
        cmd,
        cwd=root / "conformance",
        text=True,
        capture_output=True,
        check=False,
    )
    return {
        "ok": proc.returncode == 0,
        "stdout": proc.stdout,
        "stderr": proc.stderr,
        "returncode": proc.returncode,
    }


def run_w3a_trust(package_root: str, w3a_path: str) -> dict[str, Any]:
    root = Path(package_root)
    proc = subprocess.run(
        ["go", "run", "./tools/w3a", "trust", str(Path(w3a_path))],
        cwd=root / "conformance",
        text=True,
        capture_output=True,
        check=False,
    )
    if proc.returncode != 0:
        return {"ok": False, "stderr": proc.stderr, "returncode": proc.returncode}
    return {"ok": True, "trust": json.loads(proc.stdout)}
