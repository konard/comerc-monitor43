#!/usr/bin/env python3
"""
Simulate the output of `task bdd-pending` based on static analysis.

Since the actual godog run requires Docker / testcontainers, we approximate
what `task bdd-pending` would emit by checking every step line in
features/**/*.feature against the regex registered for that epic via
runner.go's switch case.

A step is "undefined" if no registered regex matches.
A step is "pending" if its handler returns godog.ErrPending. We grep the
go source for `godog.ErrPending` to detect those.

Print the simulated `out.txt` lines: "<status>\\t<file>:<line>".
"""
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
SUITE_DIR = ROOT / "features" / "suite"
FEATURES_DIR = ROOT / "features"

STEP_KEYWORDS = ("Given", "When", "Then", "And", "But", "*",
                 "Дано", "Когда", "Тогда", "И", "А")

EXISTING_REGEX_RE = re.compile(r"ctx\.Step\(\s*`([^`]*)`")


def strip_step_keyword(text: str) -> str:
    s = text.lstrip()
    for kw in STEP_KEYWORDS:
        if s.startswith(kw + " "):
            return s[len(kw) + 1:]
        if s == kw:
            return ""
    return s


def collect_feature_steps():
    for feature_dir in sorted(FEATURES_DIR.glob("[0-9][0-9]_*")):
        epic = feature_dir.name
        for feat in sorted(feature_dir.glob("*.feature")):
            in_doc = False
            for i, line in enumerate(feat.read_text(encoding="utf-8").splitlines(), 1):
                stripped = line.strip()
                if not stripped:
                    continue
                if stripped.startswith('"""'):
                    in_doc = not in_doc
                    continue
                if in_doc:
                    continue
                if stripped.startswith("|"):
                    continue
                if stripped.startswith("#") or stripped.startswith("@"):
                    continue
                first = stripped.split(maxsplit=1)
                if not first:
                    continue
                if first[0] in STEP_KEYWORDS:
                    text = strip_step_keyword(stripped)
                    if not text:
                        continue
                    yield (epic, feat, i, text)


def parse_funcs_in_file(path: Path):
    text = path.read_text(encoding="utf-8")
    i = 0
    while True:
        idx = text.find("\nfunc ", i)
        if idx == -1:
            return
        j = idx + len("\nfunc ")
        if j < len(text) and text[j] == "(":
            close_recv = text.find(")", j)
            if close_recv == -1:
                return
            j = close_recv + 1
            while j < len(text) and text[j] == " ":
                j += 1
        name_match = re.match(r"(\w+)", text[j:])
        if not name_match:
            i = idx + 1
            continue
        fname = name_match.group(1)
        body_start = text.find("{", j)
        if body_start == -1:
            return
        depth = 0
        k = body_start
        while k < len(text):
            c = text[k]
            if c == "{":
                depth += 1
            elif c == "}":
                depth -= 1
                if depth == 0:
                    break
            k += 1
        body = text[body_start:k]
        for sm in EXISTING_REGEX_RE.finditer(body):
            yield (fname, sm.group(1))
        i = k


def parse_runner_epic_funcs():
    runner = (SUITE_DIR / "runner.go").read_text(encoding="utf-8")
    epic_funcs = {}
    case_re = re.compile(
        r"case len\(epicName\) >= 2 && epicName\[:2\] == \"(\d+)\":(.*?)(?=\n\tcase |\n\tdefault:)",
        re.S,
    )
    for m in case_re.finditer(runner):
        epic_num = m.group(1)
        body = m.group(2)
        funcs = []
        for fm in re.finditer(r"\b(Register\w+)\(", body):
            funcs.append(fm.group(1))
        epic_funcs[epic_num] = funcs
    return epic_funcs


def build_per_epic_existing():
    func_to_regex = {}
    for path in sorted(SUITE_DIR.glob("steps_*.go")):
        for fname, regex in parse_funcs_in_file(path):
            func_to_regex.setdefault(fname, []).append(regex)
    epic_funcs = parse_runner_epic_funcs()
    epic_patterns = {}
    for epic_num, funcs in epic_funcs.items():
        compiled = []
        for f in funcs:
            for r in func_to_regex.get(f, []):
                try:
                    compiled.append(re.compile(r))
                except re.error:
                    pass
        epic_patterns[epic_num] = compiled
    return epic_patterns


def detect_pending_handlers():
    """Return True if any step file contains godog.ErrPending."""
    for path in SUITE_DIR.glob("steps_*.go"):
        text = path.read_text(encoding="utf-8")
        if re.search(r"\bgodog\.ErrPending\b", text):
            return True
    return False


def main():
    epic_patterns = build_per_epic_existing()
    pending_present = detect_pending_handlers()

    undefined = []
    for epic, feat, lineno, text in collect_feature_steps():
        epic_num = epic[:2]
        compiled = epic_patterns.get(epic_num, [])
        if not any(p.search(text) for p in compiled):
            rel = feat.relative_to(ROOT)
            undefined.append(("undefined", str(rel), lineno))

    print(f"# Simulated out.txt entries:")
    print(f"#   undefined: {len(undefined)}")
    print(f"#   pending: {'unknown' if pending_present else '0 (no godog.ErrPending in step files)'}")
    print()
    for status, rel, lineno in undefined[:50]:
        print(f"{status}\t{rel}:{lineno}")
    if len(undefined) > 50:
        print(f"... and {len(undefined) - 50} more undefined entries")


if __name__ == "__main__":
    main()
