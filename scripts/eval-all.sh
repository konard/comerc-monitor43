#!/usr/bin/env bash
set -euo pipefail

# Получаем команду из аргументов
COMMAND="$1"

# Выполняем поиск модулей и запуск переданной команды в каждом
modules="$(sed -n 's/^[[:space:]]*\(\.\/[^[:space:]]*\)[[:space:]]*$/\1/p' go.work)"

for dir in $modules; do
  echo "==> ${dir}"
  (
    cd "$dir"
    # Выполняем команду, переданную в скрипт
    eval "$COMMAND"
  )
done

echo "==> Total time: $SECONDS sec."