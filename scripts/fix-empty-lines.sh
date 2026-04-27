#!/usr/bin/env bash

###############################################################################
# Fix Empty Lines Between Scenarios
# Adds empty lines after each scenario (Gherkin best practice)
###############################################################################

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FEATURE_DIR="${SCRIPT_DIR}/../features"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Fix Empty Lines Between Scenarios${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

cd "$FEATURE_DIR"

# Count total scenarios
total_scenarios=$(find . -name "*.feature" -type f -exec grep -c "^  Scenario:" {} \; | awk '{s+=$1} END {print s}')

echo "Found $total_scenarios scenarios to process"
echo ""

# Add empty line after each scenario line
# This uses sed to append a newline after lines matching "  Scenario:"
find . -name "*.feature" -type f -exec sed -i '' '/^  Scenario:/a\
' {} \;

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Added empty lines after $total_scenarios scenarios${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Check changes with: git diff features/"
echo "Revert with: git checkout -- features/"
