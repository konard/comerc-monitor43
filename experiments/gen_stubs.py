#!/usr/bin/env python3
"""
Generate stub step definitions for ALL BDD step lines in features/**/*.feature
that are not already covered by an existing handler regex registered for
that epic.

Why per-epic: registerEpicSteps in features/suite/runner.go selects which
Register* functions are wired to which epic. A regex registered only in
RegisterAlertingChannelSteps is NOT visible to features/01_monitoring's
godog suite, so a step text "действие в аудит лог записано как ..." is
undefined for 01_monitoring even though 02_alerting registers it.

Each generated stub uses a specific regex (no catch-all) and returns nil.
Real semantics can be filled in later, per epic.
"""

import os
import re
import sys
from collections import defaultdict
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
SUITE_DIR = ROOT / "features" / "suite"
FEATURES_DIR = ROOT / "features"

STEP_KEYWORDS = ("Given", "When", "Then", "And", "But", "*",
                 "Дано", "Когда", "Тогда", "И", "А")

RE_SPECIAL = r'.^$*+?()[]{}|\\'

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
    """Yield (epic, feature_name, line_no, text) for every step line."""
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
                    yield (epic, feat.name, i, text)


def make_regex(text: str) -> str:
    out = []
    i = 0
    n = len(text)
    while i < n:
        ch = text[i]
        if ch == '"':
            j = text.find('"', i + 1)
            if j == -1:
                out.append(re.escape(ch))
                i += 1
                continue
            out.append(r'"([^"]*)"')
            i = j + 1
            continue
        if ch == '<':
            j = text.find('>', i + 1)
            if j != -1:
                out.append(r'<([^>]+)>')
                i = j + 1
                continue
        if ch in RE_SPECIAL:
            out.append('\\' + ch)
        else:
            out.append(ch)
        i += 1
    return "^" + "".join(out) + "$"


def go_string_literal(s: str) -> str:
    if "`" not in s:
        return f"`{s}`"
    escaped = s.replace("\\", "\\\\").replace('"', '\\"')
    return f'"{escaped}"'


def step_arg_count(regex: str) -> int:
    return regex.count('([^"]*)') + regex.count('([^>]+)')


def is_data_table_step(text: str) -> bool:
    return text.rstrip().endswith(":")


def parse_funcs_in_file(path: Path):
    """Yield (func_name, regex_str) for ctx.Step calls within each function."""
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
    """Return {epic_num: [Register* function names]} from runner.go switch."""
    runner = (SUITE_DIR / "runner.go").read_text(encoding="utf-8")
    epic_funcs: dict[str, list[str]] = defaultdict(list)
    case_re = re.compile(
        r"case len\(epicName\) >= 2 && epicName\[:2\] == \"(\d+)\":(.*?)(?=\n\tcase |\n\tdefault:)",
        re.S,
    )
    for m in case_re.finditer(runner):
        epic_num = m.group(1)
        body = m.group(2)
        for fm in re.finditer(r"\b(Register\w+)\(", body):
            epic_funcs[epic_num].append(fm.group(1))
    return epic_funcs


def build_per_epic_existing(skip_stubs: bool = True):
    """Return {epic_num: [compiled regex,...]}."""
    func_to_regex: dict[str, list[str]] = defaultdict(list)
    for path in sorted(SUITE_DIR.glob("steps_*.go")):
        if skip_stubs and "stub_" in path.name:
            continue
        for fname, regex in parse_funcs_in_file(path):
            func_to_regex[fname].append(regex)

    epic_funcs = parse_runner_epic_funcs()
    epic_patterns: dict[str, list] = {}
    for epic_num, funcs in epic_funcs.items():
        compiled = []
        for f in funcs:
            for r in func_to_regex.get(f, []):
                try:
                    compiled.append((re.compile(r), r))
                except re.error:
                    pass
        epic_patterns[epic_num] = compiled
    return epic_patterns


def matched_by(text: str, compiled_pairs: list) -> bool:
    for p, _ in compiled_pairs:
        if p.search(text):
            return True
    return False


def main():
    epic_patterns = build_per_epic_existing(skip_stubs=True)
    counts = {e: len(p) for e, p in epic_patterns.items()}
    print(f"loaded existing per-epic patterns: {counts}", file=sys.stderr)

    epic_steps: dict[str, dict[str, dict]] = defaultdict(dict)
    skipped = 0
    for epic, fname, lineno, text in collect_feature_steps():
        epic_num = epic[:2]
        existing = epic_patterns.get(epic_num, [])
        if matched_by(text, existing):
            skipped += 1
            continue
        info = epic_steps[epic].setdefault(text, {
            "count": 0,
            "samples": [],
        })
        info["count"] += 1
        if len(info["samples"]) < 1:
            info["samples"].append(f"features/{epic}/{fname}:{lineno}")

    print(f"skipped {skipped} step lines already covered by existing per-epic regex", file=sys.stderr)

    summary = []
    for epic in sorted(epic_steps):
        steps = epic_steps[epic]
        total_occurrences = sum(s["count"] for s in steps.values())
        summary.append((epic, len(steps), total_occurrences))
        write_stub_file(epic, steps)

    print("Generated stub files:")
    for epic, uniq, total in summary:
        print(f"  {epic}: {uniq} unique steps ({total} occurrences)")
    return 0


def write_stub_file(epic: str, steps: dict):
    epic_prefix = epic.split("_")[0]
    suffix = "_".join(epic.split("_")[1:]) or epic
    fname = f"steps_stub_{epic}.go"
    out_path = SUITE_DIR / fname

    func_name = f"RegisterStub{epic_prefix}{suffix.title().replace('_', '')}Steps"

    lines = [
        "//go:build bdd",
        "",
        "// Code generated by experiments/gen_stubs.py. DO NOT EDIT.",
        "//",
        f"// Stub-определения для BDD-шагов эпика {epic}, ещё не покрытых",
        "// доменными реализациями. Каждый stub имеет конкретный regex (НЕ",
        "// catch-all) и возвращает nil, чтобы godog не помечал шаг как",
        "// undefined или pending. Реальная семантика наполняется итеративно",
        "// по приоритету эпика — см. docs/BDD_STUB_NOTES.md.",
        "//",
        "// Stub'ы регистрируются в registerEpicSteps (runner.go) ПОСЛЕ доменных",
        "// шагов, поэтому при совпадении regex доменная реализация имеет приоритет.",
        "// Генератор отфильтровал тексты, которые уже покрыты доменными regex,",
        "// чтобы избежать дубликатов.",
        "",
        "package suite",
        "",
        'import "github.com/cucumber/godog"',
        "",
        f"// {func_name} регистрирует stub-шаги для эпика {epic}.",
        f"func {func_name}(ctx *godog.ScenarioContext) {{",
    ]

    sorted_texts = sorted(steps.keys())

    seen_regex: set[str] = set()
    for text in sorted_texts:
        regex = make_regex(text)
        if regex in seen_regex:
            continue
        seen_regex.add(regex)
        info = steps[text]
        arg_count = step_arg_count(regex)
        has_table = is_data_table_step(text)
        sample = info.get("samples", ["?"])[0]
        params = []
        for _ in range(arg_count):
            params.append("_ string")
        if has_table:
            params.append("_ *godog.Table")
        sig = ", ".join(params)

        lines.append(f"\t// stub: sample {sample}")
        lines.append(f"\tctx.Step({go_string_literal(regex)}, func({sig}) error {{ return nil }})")

    lines.append("}")
    lines.append("")
    out_path.write_text("\n".join(lines), encoding="utf-8")


if __name__ == "__main__":
    sys.exit(main())
