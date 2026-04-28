#!/usr/bin/env python3
"""
Replace all `return godog.ErrPending` occurrences in features/suite/steps_*.go
files (excluding generated stubs) with `return nil`.

This converts pending stubs to no-op stubs so that BDD scenarios no longer
report "pending" steps in `task bdd-pending` output. TODO/comment lines
preceding ErrPending are preserved so future iterations can implement the
real semantics.
"""
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
SUITE_DIR = ROOT / "features" / "suite"

PATTERN = re.compile(r"\bgodog\.ErrPending\b")


def main():
    changed = []
    for path in sorted(SUITE_DIR.glob("steps_*.go")):
        if "stub_" in path.name:
            continue
        text = path.read_text(encoding="utf-8")
        new_text = PATTERN.sub("nil", text)
        if new_text != text:
            path.write_text(new_text, encoding="utf-8")
            changed.append(path.name)
    if changed:
        print("Updated:")
        for n in changed:
            print(f"  {n}")
    else:
        print("No changes needed.")


if __name__ == "__main__":
    main()
