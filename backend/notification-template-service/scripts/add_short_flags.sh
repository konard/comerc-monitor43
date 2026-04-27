#!/bin/bash
# Скрипт для добавления -short флага во все integration тесты

file="internal/repository/postgres_template_repository_test.go"

# Добавляем -short проверку после каждого "func Test"
awk '/^func Test.*\(t \*testing.T\) \{/ {
    print
    print "\tif testing.Short() {"
    print "\t\tt.Skip(\"skipping integration test in short mode\")"
    print "\t}"
    print ""
    next
}
{ print }
' "$file" > "${file}.tmp" && mv "${file}.tmp" "$file"

echo "Added -short flags to $file"
