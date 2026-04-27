#!/usr/bin/env bash
# Собирает стандартный диагностический снимок окружения.
# Аргументы: [port...]
# Печатает git/go/docker/env/ports для ручного анализа.
# Ничего не изменяет.

set -euo pipefail

ports=("$@")
if [ "${#ports[@]}" -eq 0 ]; then
  ports=(5432 5672 9000 9092)
fi

section() {
  printf '\n## %s\n' "$1"
}

run_or_note() {
  local name="$1"
  shift

  if command -v "$1" >/dev/null 2>&1; then
    "$@"
  else
    printf '%s not available\n' "$name"
  fi
}

section "Context"
pwd
printf 'date: %s\n' "$(date -Is 2>/dev/null || date)"

section "Git"
run_or_note git git status --short

section "Go"
run_or_note go go env GOMOD GOVERSION GOPATH

section "Docker"
run_or_note docker docker ps

section "Env Files"
find . -maxdepth 2 \( -name '.env' -o -name '.env.*' \) -print 2>/dev/null || true

section "Ports"
for port in "${ports[@]}"; do
  printf '\n### %s\n' "$port"
  if command -v lsof >/dev/null 2>&1; then
    lsof -nP -i :"$port" || true
  else
    printf 'lsof not available\n'
  fi
done
