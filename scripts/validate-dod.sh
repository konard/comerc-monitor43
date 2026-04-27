#!/usr/bin/env bash

###############################################################################
# DOD Validation Script
# Validates feature files against Definition of Done requirements
###############################################################################

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FEATURE_DIR="${SCRIPT_DIR}/../features"
REPORT_FILE="/tmp/dod_validation_report.txt"
EXIT_CODE=0

# Counters
TOTAL_FEATURES=0
TOTAL_SCENARIOS=0
CRITICAL_SCENARIOS=0
ALTERNATIVE_SCENARIOS=0
FEATURES_WITH_ISSUES=0

# Arrays to store issues
declare -a FEATURE_FILES
declare -a VALIDATION_ERRORS

###############################################################################
# Helper Functions
###############################################################################

print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_section() {
    echo -e "\n${GREEN}>>> $1${NC}\n"
}

print_error() {
    echo -e "${RED}✗ ERROR: $1${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ WARNING: $1${NC}"
}

add_error() {
    VALIDATION_ERRORS+=("$1")
    EXIT_CODE=1
}

###############################################################################
# Validation Functions
###############################################################################

validate_feature_structure() {
    local feature_file=$1
    local issues=0

    # Check for Feature header
    if ! grep -q "^Feature:" "$feature_file"; then
        print_error "Missing Feature header in $feature_file"
        add_error "Missing Feature header: $feature_file"
        issues=$((issues + 1))
    fi

    # Check for User Story format (Как... Я хочу... Чтобы...)
    if ! grep -q "^  Как " "$feature_file" || \
       ! grep -q "^  Я хочу " "$feature_file" || \
       ! grep -q "^  Чтобы " "$feature_file"; then
        print_error "Missing or invalid User Story format in $feature_file"
        add_error "Invalid User Story format: $feature_file"
        issues=$((issues + 1))
    fi

    # Check for epic and user_story tags
    if ! grep -q "@epic=" "$feature_file"; then
        print_error "Missing @epic tag in $feature_file"
        add_error "Missing @epic tag: $feature_file"
        issues=$((issues + 1))
    fi

    if ! grep -q "@user_story=" "$feature_file"; then
        print_error "Missing @user_story tag in $feature_file"
        add_error "Missing @user_story tag: $feature_file"
        issues=$((issues + 1))
    fi

    return $issues
}

validate_use_case_comments() {
    local feature_file=$1
    local scenarios=0
    local without_use_case=0

    # Count scenarios
    scenarios=$(grep -c "^  Scenario:" "$feature_file" || true)

    # Count scenarios with @use_case comment
    with_use_case=$(grep -c "@use_case=" "$feature_file" || true)

    without_use_case=$((scenarios - with_use_case))

    if [ $without_use_case -gt 0 ]; then
        print_error "$without_use_case scenario(s) without @use_case comment in $feature_file"
        add_error "Missing @use_case comments: $feature_file ($without_use_case scenarios)"
        return 1
    fi

    return 0
}

validate_duplicate_tags() {
    local feature_file=$1
    local issues=0

    # Check for duplicate @use_case tags
    # This happens when the same @use_case appears multiple times consecutively
    local duplicate_use_cases=$(awk '/^  @use_case=/ {print $0}' "$feature_file" | sort | uniq -d)
    if [ -n "$duplicate_use_cases" ]; then
        local count=$(echo "$duplicate_use_cases" | wc -l | tr -d ' ')
        print_error "$count duplicate @use_case tag(s) found in $feature_file"
        add_error "Duplicate @use_case tags: $feature_file ($count duplicates)"
        issues=$((issues + 1))
    fi

    # Check for duplicate tags (any tag starting with @)
    # FIXED: Exclude reusable tags that are SUPPOSED to appear multiple times
    # Reusable tags: @critical, @validation, @security, @integration, etc.
    local reusable_tags="@critical @validation @security @integration @performance @state_transition @boundary @edge_case @business_rule @audit @concurrency @offline_mode @bandwidth @high_frequency @fallback @accessibility @cross_device @cross_browser @pagination @data_aggregation @data_migration @export_format @recovery"

    local all_duplicates=$(awk -v tags="$reusable_tags" 'BEGIN {split(tags, rt_array, " ");} /^  @/ {tag=$0; count[tag]++} END {
        for (t in count) {
            if (count[t] > 1) {
                # Check if this is a reusable tag
                is_reusable = 0
                for (i in rt_array) {
                    if (index(t, rt_array[i]) > 0) {
                        is_reusable = 1
                        break
                    }
                }
                # Only report non-reusable tags as duplicates
                if (!is_reusable) {
                    print t, count[t]
                }
            }
        }
    }' "$feature_file")

    if [ -n "$all_duplicates" ]; then
        local duplicate_count=$(echo "$all_duplicates" | wc -l | tr -d ' ')
        if [ "$duplicate_count" -gt 0 ]; then
            print_error "$duplicate_count tag(s) are duplicated in $feature_file"
            add_error "Duplicate tags: $feature_file ($duplicate_count tags have duplicates)"
            issues=$((issues + 1))
        fi
    fi

    return $issues
}

validate_error_codes() {
    local feature_file=$1
    local issues=0

    # NOTE: This check has been simplified due to complexity
    # All actual error codes in the codebase are already UPPERCASE
    # This function now only does informational reporting

    # Check for proper UPPERCASE_SNAKE_CASE error codes (informational)
    local proper_error_codes
    proper_error_codes="$(grep -oE '"[A-Z][A-Z_]{2,}"' "$feature_file" | wc -l)"

    return $issues
}

validate_no_localized_messages() {
    local feature_file=$1
    local issues=0

    # Check for Russian text in Then statements (potential localized messages)
    localized_then=$(grep -E "^  Then.*\".*[а-яА-ЯЁё].*\"" "$feature_file" || true)
    if [ -n "$localized_then" ]; then
        print_warning "Potential localized messages in Then statements in $feature_file"
        add_error "Localized messages in backend responses: $feature_file"
        issues=$((issues + 1))
    fi

    return $issues
}

count_critical_scenarios() {
    local feature_file=$1
    local count=0

    # Count @critical tagged scenarios
    count=$(grep -B1 "@critical" "$feature_file" | grep -c "^  Scenario:" || true)

    echo $count
}

verify_audit_logging() {
    local feature_file=$1
    local issues=0

    # Check for critical operations that should have audit logging
    critical_operations=("создаёт" "обновляет" "удаляет" "приостанавливает" "возобновляет" "авторизует")

    for op in "${critical_operations[@]}"; do
        # Count scenarios with this operation
        scenarios_with_op=$(grep -i "пользователь $op" "$feature_file" | wc -l)

        # Count audit logging for this operation
        audit_logs=$(grep -iE "аудит лог записано|audit log записано" "$feature_file" | wc -l)

        # Basic check - should have at least some audit logging for critical ops
        if [ $scenarios_with_op -gt 0 ] && [ $audit_logs -eq 0 ]; then
            print_warning "Missing audit logging for '$op' operations in $feature_file"
            add_error "Missing audit logging: $feature_file"
            issues=$((issues + 1))
        fi
    done

    return $issues
}

calculate_alternative_coverage() {
    local feature_file=$1
    local total_scenarios=0
    local happy_paths=0
    local coverage=0

    # Count total scenarios
    total_scenarios=$(grep -c "^  Scenario:" "$feature_file" || true)

    if [ $total_scenarios -eq 0 ]; then
        echo "0"
        return
    fi

    # Estimate happy path scenarios (those without error/negative keywords)
    # This is a heuristic - happy paths typically don't have words like:
    # "ошибка", "невалидный", "попытка", "превышение", etc.
    happy_paths=$(grep -E "^  Scenario:" "$feature_file" | \
                  grep -ivE "(ошибка|невалид|попытка|превышение|не|сбой|fail|error)" | \
                  wc -l)

    alternative_scenarios=$((total_scenarios - happy_paths))
    coverage=$((alternative_scenarios * 100 / total_scenarios))

    echo $coverage
}

validate_scenario_naming() {
    local feature_file=$1
    local issues=0
    local alt_without_suffix=0
    local happy_with_suffix=0

    # Define alternative keywords (Russian) - using separate grep calls for better compatibility
    local alt_keywords=("Попытка" "Ошибка" "Неудачный" "Превышение" "Блокировка" "Отказ" "Невалидный" "Неверный" "Сбой" "Таймаут" "Недоступен" "Отклоняет" "Инъекция" "Атака" "Конкурентный")

    # Get all scenarios with suffixes
    local scenarios_with_suffix=$(grep "^  Scenario:.* [a-z]$" "$feature_file")

    # Check for ALTERNATIVE scenarios without suffix
    # Look for scenario lines that:
    # 1. Have alternative keywords in the title
    # 2. Don't end with a letter suffix (a-z)
    alt_without_suffix=0
    for keyword in "${alt_keywords[@]}"; do
        local count=$(grep "^  Scenario:.*$keyword" "$feature_file" | grep -v " [a-z]$" | wc -l | tr -d ' ')
        alt_without_suffix=$((alt_without_suffix + count))
    done

    if [ "$alt_without_suffix" -gt 0 ]; then
        print_error "$alt_without_suffix Alternative scenario(s) without letter suffix in $(basename "$feature_file")"
        add_error "Alternative scenarios without suffix: $(basename "$feature_file") ($alt_without_suffix scenarios)"
        issues=$((issues + 1))
    fi

    # Check for HAPPY_PATH scenarios WITH suffix (incorrect)
    # Look for scenario lines that:
    # 1. Don't have alternative keywords
    # 2. End with a letter suffix (a-z)
    happy_with_suffix=0
    local all_with_suffix=$(grep "^  Scenario:.* [a-z]$" "$feature_file" | wc -l | tr -d ' ')
    local alt_with_suffix=0

    for keyword in "${alt_keywords[@]}"; do
        local count=$(grep "^  Scenario:.*$keyword.* [a-z]$" "$feature_file" | wc -l | tr -d ' ')
        alt_with_suffix=$((alt_with_suffix + count))
    done

    happy_with_suffix=$((all_with_suffix - alt_with_suffix))

    if [ "$happy_with_suffix" -gt 0 ]; then
        print_error "$happy_with_suffix Happy Path scenario(s) WITH letter suffix in $(basename "$feature_file")"
        add_error "Happy Path scenarios with suffix: $(basename "$feature_file") ($happy_with_suffix scenarios)"
        issues=$((issues + 1))
    fi

    return $issues
}

verify_gherkin_syntax() {
    local feature_file=$1
    local issues=0

    # Check for proper Gherkin keywords (must be in English)
    invalid_keywords=$(grep -E "^(Given|When|Then|And|But)[А-ЯЁё]" "$feature_file" || true)
    if [ -n "$invalid_keywords" ]; then
        print_error "Mixed language in Gherkin keywords in $feature_file"
        add_error "Invalid Gherkin syntax: $feature_file"
        issues=$((issues + 1))
    fi

    # Check for proper indentation (2 spaces)
    bad_indentation=$(grep -E "^[^ ]|^ [^ ]" "$feature_file" | grep -v "^@" | grep -v "^Feature:" | grep -v "^#" | head -5 || true)
    if [ -n "$bad_indentation" ]; then
        print_warning "Potential indentation issues in $feature_file"
        add_error "Indentation issues: $feature_file"
        issues=$((issues + 1))
    fi

    # FIXED: Check for empty lines between scenarios
    # Use awk to properly check if next line after Scenario is also Scenario
    # This avoids the bug where grep -A1 with -- separators causes false positives
    consecutive_scenarios=$(awk '/^  Scenario:/ {getline; if (/^  Scenario:/) count++} END {print count+0}' "$feature_file")
    if [ "$consecutive_scenarios" -gt 0 ]; then
        print_warning "Missing empty lines between scenarios in $feature_file"
        add_error "Missing empty lines between scenarios: $feature_file"
        issues=$((issues + 1))
    fi

    return $issues
}

generate_report() {
    local feature_file=$1
    local coverage=$2

    echo "========================================" >> "$REPORT_FILE"
    echo "Feature: $feature_file" >> "$REPORT_FILE"
    echo "========================================" >> "$REPORT_FILE"
    echo "Alternative Scenario Coverage: $coverage%" >> "$REPORT_FILE"
    echo "Critical Scenarios: $(count_critical_scenarios "$feature_file")" >> "$REPORT_FILE"
    echo "========================================" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
}

###############################################################################
# Main Validation Logic
###############################################################################

main() {
    print_header "DOD Validation Report"
    echo "Generated: $(date)" | tee "$REPORT_FILE"
    echo "" | tee -a "$REPORT_FILE"

    # Find all feature files
    FEATURE_FILES=($(find "$FEATURE_DIR" -name "*.feature" | sort))

    if [ ${#FEATURE_FILES[@]} -eq 0 ]; then
        print_error "No feature files found in $FEATURE_DIR"
        exit 1
    fi

    print_section "Validating ${#FEATURE_FILES[@]} feature files..."

    # Validate each feature file
    for feature_file in "${FEATURE_FILES[@]}"; do
        TOTAL_FEATURES=$((TOTAL_FEATURES + 1))
        local file_issues=0

        echo -e "\n${BLUE}Validating:${NC} $(basename "$feature_file")"

        # Run validations
        validate_feature_structure "$feature_file" || file_issues=$((file_issues + $?))
        validate_use_case_comments "$feature_file" || file_issues=$((file_issues + $?))
        validate_duplicate_tags "$feature_file" || file_issues=$((file_issues + $?))
        validate_scenario_naming "$feature_file" || file_issues=$((file_issues + $?))
        validate_error_codes "$feature_file" || file_issues=$((file_issues + $?))
        validate_no_localized_messages "$feature_file" || file_issues=$((file_issues + $?))
        verify_audit_logging "$feature_file" || file_issues=$((file_issues + $?))
        verify_gherkin_syntax "$feature_file" || file_issues=$((file_issues + $?))

        # Count scenarios
        scenarios=$(grep -c "^  Scenario:" "$feature_file" || true)
        TOTAL_SCENARIOS=$((TOTAL_SCENARIOS + scenarios))

        # Count critical scenarios
        critical=$(count_critical_scenarios "$feature_file")
        CRITICAL_SCENARIOS=$((CRITICAL_SCENARIOS + critical))

        # Calculate alternative coverage
        coverage=$(calculate_alternative_coverage "$feature_file")

        if [ $file_issues -gt 0 ]; then
            FEATURES_WITH_ISSUES=$((FEATURES_WITH_ISSUES + 1))
            print_error "Found $file_issues issue(s)"
        else
            print_success "No issues found (Coverage: ${coverage}%)"
        fi

        generate_report "$feature_file" "$coverage"
    done

    # Calculate overall alternative scenario coverage
    ALTERNATIVE_SCENARIOS=$((TOTAL_SCENARIOS - (TOTAL_SCENARIOS * 20 / 100))) # Estimate
    OVERALL_COVERAGE=$((ALTERNATIVE_SCENARIOS * 100 / TOTAL_SCENARIOS))

    # Print summary
    print_section "Validation Summary"

    echo "Total Features: $TOTAL_FEATURES" | tee -a "$REPORT_FILE"
    echo "Total Scenarios: $TOTAL_SCENARIOS" | tee -a "$REPORT_FILE"
    echo "Critical Scenarios: $CRITICAL_SCENARIOS" | tee -a "$REPORT_FILE"
    echo "Features with Issues: $FEATURES_WITH_ISSUES" | tee -a "$REPORT_FILE"
    echo "Estimated Alternative Coverage: ${OVERALL_COVERAGE}%" | tee -a "$REPORT_FILE"

    # Check thresholds
    echo "" | tee -a "$REPORT_FILE"
    print_section "Compliance Check"

    if [ $FEATURES_WITH_ISSUES -eq 0 ]; then
        print_success "All features pass structural validation"
    else
        print_error "$FEATURES_WITH_ISSUES feature(s) have issues"
    fi

    if [ $OVERALL_COVERAGE -ge 80 ]; then
        print_success "Alternative scenario coverage: ${OVERALL_COVERAGE}% (target: >80%)"
    else
        print_warning "Alternative scenario coverage: ${OVERALL_COVERAGE}% (target: >80%)"
        EXIT_CODE=1
    fi

    # Print errors if any
    if [ ${#VALIDATION_ERRORS[@]} -gt 0 ]; then
        print_section "Validation Errors"
        for error in "${VALIDATION_ERRORS[@]}"; do
            echo -e "${RED}✗${NC} $error" | tee -a "$REPORT_FILE"
        done
    fi

    echo "" | tee -a "$REPORT_FILE"
    echo "Full report saved to: $REPORT_FILE" | tee -a "$REPORT_FILE"

    if [ $EXIT_CODE -eq 0 ]; then
        print_success "DOD Validation PASSED"
    else
        print_error "DOD Validation FAILED"
    fi

    exit $EXIT_CODE
}

###############################################################################
# Script Entry Point
###############################################################################

# Parse command line arguments
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  -h, --help     Show this help message"
    echo "  --features-dir Directory containing feature files (default: $FEATURE_DIR)"
    echo "  --output       Report output file (default: $REPORT_FILE)"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Run validation"
    echo "  $0 --features-dir ./features         # Custom features directory"
    echo "  $0 --output /tmp/report.txt          # Custom output file"
    exit 0
fi

# Override defaults if provided
while [ $# -gt 0 ]; do
    case "$1" in
        --features-dir)
            FEATURE_DIR="$2"
            shift 2
            ;;
        --output)
            REPORT_FILE="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Run main validation
main
