#!/usr/bin/env bash

###############################################################################
# Fix Error Codes in Feature Files
# Replaces lowercase error codes with UPPERCASE_SNAKE_CASE
###############################################################################

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FEATURE_DIR="${SCRIPT_DIR}/../features"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Fix Error Codes in Feature Files${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

cd "$FEATURE_DIR"

# Common error codes mapping
declare -A error_codes=(
  # Basic validation errors
  ["invalid_url"]="INVALID_URL"
  ["invalid_input"]="INVALID_INPUT"
  ["invalid_name"]="INVALID_NAME"
  ["invalid_email"]="INVALID_EMAIL"
  ["invalid_interval"]="INVALID_INTERVAL"
  ["invalid_timeout"]="INVALID_TIMEOUT"
  ["invalid_threshold"]="INVALID_THRESHOLD"
  ["invalid_value"]="INVALID_VALUE"
  ["invalid_format"]="INVALID_FORMAT"
  ["invalid_parameter"]="INVALID_PARAMETER"
  ["invalid_request"]="INVALID_REQUEST"

  # Authentication & Authorization
  ["unauthorized"]="UNAUTHORIZED"
  ["forbidden"]="FORBIDDEN"
  ["authentication_failed"]="AUTHENTICATION_FAILED"
  ["access_denied"]="ACCESS_DENIED"
  ["token_expired"]="TOKEN_EXPIRED"
  ["invalid_token"]="INVALID_TOKEN"
  ["invalid_credentials"]="INVALID_CREDENTIALS"

  # Resource errors
  ["not_found"]="NOT_FOUND"
  ["already_exists"]="ALREADY_EXISTS"
  ["resource_not_found"]="RESOURCE_NOT_FOUND"
  ["duplicate_resource"]="DUPLICATE_RESOURCE"

  # Rate limiting & Quotas
  ["rate_limit_exceeded"]="RATE_LIMIT_EXCEEDED"
  ["quota_exceeded"]="QUOTA_EXCEEDED"
  ["too_many_requests"]="TOO_MANY_REQUESTS"
  ["limit_reached"]="LIMIT_REACHED"

  # Payment & Billing
  ["payment_required"]="PAYMENT_REQUIRED"
  ["payment_failed"]="PAYMENT_FAILED"
  ["insufficient_funds"]="INSUFFICIENT_FUNDS"
  ["subscription_expired"]="SUBSCRIPTION_EXPIRED"
  ["billing_error"]="BILLING_ERROR"

  # System errors
  ["internal_error"]="INTERNAL_ERROR"
  ["service_unavailable"]="SERVICE_UNAVAILABLE"
  ["timeout"]="TIMEOUT"
  ["server_error"]="SERVER_ERROR"
  ["database_error"]="DATABASE_ERROR"

  # Business logic errors
  ["operation_failed"]="OPERATION_FAILED"
  ["invalid_state"]="INVALID_STATE"
  ["conflict"]="CONFLICT"
  [" precondition_failed"]="PRECONDITION_FAILED"
)

# Count total replacements
total_replacements=0

# Replace in all .feature files
for old in "${!error_codes[@]}"; do
  new="${error_codes[$old]}"

  # Count occurrences
  count=$(find . -name "*.feature" -type f -exec grep -o "\"$old\"" {} \; | wc -l | tr -d ' ')

  if [ $count -gt 0 ]; then
    echo -e "${YELLOW}Replacing${NC} \"$old\" ${YELLOW}with${NC} \"$new\" ${YELLOW}($count occurrences)${NC}"
    find . -name "*.feature" -type f -exec sed -i '' "s/\"$old\"/\"$new\"/g" {} \;
    total_replacements=$((total_replacements + count))
  fi
done

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Total replacements: $total_replacements${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Check changes with: git diff features/"
echo "Revert with: git checkout -- features/"
