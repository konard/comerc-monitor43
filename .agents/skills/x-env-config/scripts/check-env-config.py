#!/usr/bin/env python3
# Проверяет согласованность между Config, doc.go и env-файлом.
# Аргументы: <config.go> <doc.go> [env-file]
# Проверяет только наличие и совпадение env-переменных.
# Ничего не изменяет.

"""Проверяет соответствие тегов `envconfig`, содержимого `doc.go` и опционального env-файла."""

from __future__ import annotations

import pathlib
import re
import sys


ENV_TAG_RE = re.compile(r'envconfig:"([A-Z][A-Z0-9_]*)"')
DOC_ENV_RE = re.compile(r"^\s*//\s+([A-Z][A-Z0-9_]*)\b")
ENV_FILE_RE = re.compile(r"^\s*([A-Z][A-Z0-9_]*)=")


def read_text(path: pathlib.Path) -> str:
    return path.read_text(encoding="utf-8")


def collect_tag_envs(path: pathlib.Path) -> set[str]:
    return set(ENV_TAG_RE.findall(read_text(path)))


def collect_doc_envs(path: pathlib.Path) -> set[str]:
    envs: set[str] = set()
    for line in read_text(path).splitlines():
        match = DOC_ENV_RE.match(line)
        if match:
            envs.add(match.group(1))
    return envs


def collect_env_file_vars(path: pathlib.Path) -> set[str]:
    envs: set[str] = set()
    for line in read_text(path).splitlines():
        match = ENV_FILE_RE.match(line)
        if match:
            envs.add(match.group(1))
    return envs


def main(argv: list[str]) -> int:
    if len(argv) not in {3, 4}:
        print("usage: check-env-config.py <config.go> <doc.go> [env-file]", file=sys.stderr)
        return 2

    config_path = pathlib.Path(argv[1])
    doc_path = pathlib.Path(argv[2])
    env_path = pathlib.Path(argv[3]) if len(argv) == 4 else None

    tag_envs = collect_tag_envs(config_path)
    doc_envs = collect_doc_envs(doc_path)

    missing_in_doc = sorted(tag_envs - doc_envs)
    missing_in_env_file: list[str] = []

    if env_path is not None:
        env_file_vars = collect_env_file_vars(env_path)
        missing_in_env_file = sorted(tag_envs - env_file_vars)

    if not missing_in_doc and not missing_in_env_file:
        print("env config check passed")
        return 0

    if missing_in_doc:
        print("missing in doc.go:", ", ".join(missing_in_doc), file=sys.stderr)

    if missing_in_env_file:
        print("missing in env file:", ", ".join(missing_in_env_file), file=sys.stderr)

    return 1


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
