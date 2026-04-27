#!/bin/sh
# Проверяет базовую структуру doc.go в указанном пакете.
# Аргументы: <package-dir>
# Проверяет только наличие файла и package comment.
# Ничего не создаёт и не изменяет.

set -eu

if [ "$#" -ne 1 ]; then
    echo "usage: check-doc-go.sh <package-dir>" >&2
    exit 2
fi

pkg_dir=$1
base_name=$(basename "$pkg_dir")

case "$pkg_dir" in
    cmd/*|*/cmd/*)
        echo "skip: cmd package"
        exit 0
        ;;
esac

if [ "$base_name" = "mocks" ]; then
    echo "skip: mocks package"
    exit 0
fi

doc_file=$pkg_dir/doc.go

if [ ! -f "$doc_file" ]; then
    echo "missing doc.go: $pkg_dir" >&2
    exit 1
fi

if ! rg -q '^// Package ' "$doc_file"; then
    echo "missing package comment in $doc_file" >&2
    exit 1
fi

if ! rg -q '^package ' "$doc_file"; then
    echo "missing package declaration in $doc_file" >&2
    exit 1
fi

echo "doc.go check passed: $pkg_dir"
