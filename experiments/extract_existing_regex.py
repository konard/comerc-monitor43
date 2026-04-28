#!/usr/bin/env python3
"""
Extract all existing step regex patterns from features/suite/steps_*.go
files (excluding generated stub files). Outputs one regex per line.
"""
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
SUITE_DIR = ROOT / "features" / "suite"

# Match ctx.Step(`...regex...`, ...) and capture the regex.
STEP_RE = re.compile(r"ctx\.Step\(\s*`([^`]*)`")

def main():
    out = set()
    for path in sorted(SUITE_DIR.glob("steps_*.go")):
        if "stub_" in path.name:
            continue
        text = path.read_text(encoding="utf-8")
        for m in STEP_RE.finditer(text):
            out.add(m.group(1))
    for r in sorted(out):
        print(r)

if __name__ == "__main__":
    main()
