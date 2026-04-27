#!/bin/sh
# Проверяет базовый post-check после генерации mockery.
# Аргументы: <package-dir> <interface-name> [config-path]
# Проверяет .mockery.yml, generated file и ожидаемые символы.
# Ничего не создаёт и не изменяет.

set -eu

snake_case() {
    printf '%s' "$1" | sed -E 's/([A-Z]+)([A-Z][a-z])/\1_\2/g; s/([a-z0-9])([A-Z])/\1_\2/g' | tr '[:upper:]' '[:lower:]'
}

if [ "$#" -lt 2 ] || [ "$#" -gt 3 ]; then
    echo "usage: check-mockery-targets.sh <package-dir> <interface-name> [config-path]" >&2
    exit 2
fi

pkg_dir=$1
iface=$2
config_path=${3:-.mockery.yml}

if [ ! -f "$config_path" ]; then
    echo "missing mockery config: $config_path" >&2
    exit 1
fi

if ! rg -Fq "$pkg_dir" "$config_path"; then
    echo "package not found in mockery config: $pkg_dir" >&2
    exit 1
fi

file_name=$(snake_case "$iface").go
generated_file=$pkg_dir/mocks/$file_name

if [ ! -f "$generated_file" ]; then
    echo "generated mock not found: $generated_file" >&2
    exit 1
fi

if ! rg -q "New${iface}\(" "$generated_file"; then
    echo "constructor not found in generated mock: New${iface}" >&2
    exit 1
fi

if ! rg -q 'EXPECT\(\)' "$generated_file"; then
    echo "EXPECT() not found in generated mock" >&2
    exit 1
fi

echo "mockery target check passed: $generated_file"
