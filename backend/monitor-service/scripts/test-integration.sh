#!/bin/bash
# Script to run integration tests with testcontainers

set -e

cd "$(dirname "$0")/.."

echo "Running integration tests with testcontainers..."
echo "Note: This requires Docker to be running"
echo ""

# Disable ryuk for Colima compatibility
export TESTCONTAINERS_RYUK_DISABLED=true

# Run integration tests
go test ./internal/repository/postgres -v -run TestPostgres_Integration "$@"

echo ""
echo "Integration tests completed successfully!"
