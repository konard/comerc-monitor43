#!/usr/bin/env python3
"""
Verify that every step text in features/**/*.feature is matched by exactly
ONE registered regex (across both real and stub step files).

Reports:
- texts that match no regex (would still be "undefined")
- texts that match multiple regex (would be "ambiguous")
- regex patterns that are duplicated (multiple registrations of identical pattern)
"""
import re
import sys
from pathlib import Path
from collections import defaultdict

ROOT = Path(__file__).resolve().parent.parent
SUITE_DIR = ROOT / "features" / "suite"
FEATURES_DIR = ROOT / "features"

STEP_KEYWORDS = ("Given", "When", "Then", "And", "But", "*",
                 "Дано", "Когда", "Тогда", "И", "А")

REGEX_RE = re.compile(r"ctx\.Step\(\s*`([^`]*)`")


def strip_step_keyword(text: str) -> str:
    s = text.lstrip()
    for kw in STEP_KEYWORDS:
        if s.startswith(kw + " "):
            return s[len(kw) + 1:]
        if s == kw:
            return ""
    return s


def collect_steps_in_features():
    """Yield (epic, file, lineno, text) for every step line in feature files."""
    seen_keywords = STEP_KEYWORDS
    for feature_dir in sorted(FEATURES_DIR.glob("[0-9][0-9]_*")):
        epic = feature_dir.name
        for feat in sorted(feature_dir.glob("*.feature")):
            in_table = False
            in_doc = False
            for i, line in enumerate(feat.read_text(encoding="utf-8").splitlines(), 1):
                stripped = line.strip()
                if not stripped:
                    in_table = False
                    continue
                if stripped.startswith('"""'):
                    in_doc = not in_doc
                    continue
                if in_doc:
                    continue
                if stripped.startswith("|"):
                    in_table = True
                    continue
                if stripped.startswith("#") or stripped.startswith("@"):
                    continue
                # Step lines start with one of the keywords.
                first = stripped.split(maxsplit=1)
                if not first:
                    continue
                if first[0] in seen_keywords:
                    text = strip_step_keyword(stripped)
                    if not text:
                        continue
                    yield (epic, feat, i, text)


def collect_registered_patterns_per_epic():
    """Return {epic: [(regex_str, file, line)]}.

    Heuristic: stub_NN_* file targets epic NN. Other steps_*.go files target
    the epic in their filename or are common (steps_common.go).
    Register also based on runner.go's switch to know which Register* are
    called per epic.
    """
    # Parse runner.go to find which functions are called for each epic.
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

    # Map function name -> list of regex it registers.
    func_regex: dict[str, list[tuple[str, Path, int]]] = defaultdict(list)
    for path in sorted(SUITE_DIR.glob("*.go")):
        text = path.read_text(encoding="utf-8")
        # Find functions and parse out ctx.Step regex inside their body.
        # Simplistic: split by "func ".
        # Use a state machine to find function boundaries.
        i = 0
        while True:
            idx = text.find("\nfunc ", i)
            if idx == -1:
                break
            # function name
            j = idx + len("\nfunc ")
            # Possibly receiver: func (x *T) Name(... or func Name(...
            if text[j] == "(":
                # skip receiver
                close_recv = text.find(")", j)
                if close_recv == -1:
                    break
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
                break
            # find matching closing brace
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
            # capture all ctx.Step regex
            for sm in REGEX_RE.finditer(body):
                line_no = text.count("\n", 0, body_start + sm.start()) + 1
                func_regex[fname].append((sm.group(1), path, line_no))
            i = k

    # Build epic -> list of registered patterns.
    epic_patterns: dict[str, list[tuple[str, Path, int]]] = defaultdict(list)
    for epic, funcs in epic_funcs.items():
        for f in funcs:
            for pat, path, line_no in func_regex.get(f, []):
                epic_patterns[epic].append((pat, path, line_no))

    return epic_patterns


def main():
    epic_patterns = collect_registered_patterns_per_epic()
    # Pre-compile patterns for matching.
    epic_compiled: dict[str, list[tuple[re.Pattern, str, Path, int]]] = {}
    for epic, items in epic_patterns.items():
        compiled = []
        for pat, path, line_no in items:
            try:
                compiled.append((re.compile(pat), pat, path, line_no))
            except re.error as e:
                print(f"!!! invalid regex in {path}:{line_no}: {pat!r}: {e}", file=sys.stderr)
        epic_compiled[epic] = compiled

    # Walk all feature step lines and check matches per epic.
    undefined = defaultdict(list)
    ambiguous = defaultdict(list)
    matched = 0
    for epic, feat, lineno, text in collect_steps_in_features():
        epic_num = epic[:2]
        compiled = epic_compiled.get(epic_num, [])
        matches = [c for c in compiled if c[0].search(text)]
        if not matches:
            undefined[epic].append((feat.name, lineno, text))
        elif len(matches) > 1:
            # Only consider it ambiguous if patterns differ
            distinct = {m[1] for m in matches}
            if len(distinct) > 1:
                ambiguous[epic].append((feat.name, lineno, text, distinct))
            else:
                # Same regex registered multiple times - duplicate
                ambiguous[epic].append((feat.name, lineno, text, distinct))
        else:
            matched += 1

    total = matched + sum(len(v) for v in undefined.values()) + sum(len(v) for v in ambiguous.values())
    print(f"Total feature step lines: {total}")
    print(f"Matched (single):         {matched}")
    print(f"Undefined:                {sum(len(v) for v in undefined.values())}")
    print(f"Ambiguous/duplicate:      {sum(len(v) for v in ambiguous.values())}")

    if undefined:
        print("\n=== Undefined steps per epic ===")
        for epic in sorted(undefined):
            print(f"\n{epic} ({len(undefined[epic])}):")
            for fname, lineno, text in undefined[epic][:10]:
                print(f"  {fname}:{lineno}  {text}")
            if len(undefined[epic]) > 10:
                print(f"  ... and {len(undefined[epic]) - 10} more")

    if ambiguous:
        print("\n=== Ambiguous steps per epic ===")
        for epic in sorted(ambiguous):
            print(f"\n{epic} ({len(ambiguous[epic])}):")
            for fname, lineno, text, pats in ambiguous[epic][:5]:
                print(f"  {fname}:{lineno}  {text}")
                for p in list(pats)[:3]:
                    print(f"    matched by: {p}")
            if len(ambiguous[epic]) > 5:
                print(f"  ... and {len(ambiguous[epic]) - 5} more")


if __name__ == "__main__":
    main()
