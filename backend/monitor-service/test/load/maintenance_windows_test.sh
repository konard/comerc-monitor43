#!/bin/bash
# Load testing script for Maintenance Window Service
# Uses vegeta for HTTP load testing

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVICE_URL="${SERVICE_URL:-http://localhost:9090}"
DURATION="${DURATION:-30s}"
RATE="${RATE:-10}"

echo "=== Maintenance Window Load Testing ==="
echo "Service URL: $SERVICE_URL"
echo "Duration: $DURATION"
echo "Rate: $RATE requests/second"
echo ""

# Check if service is running
echo "Checking service availability..."
if ! curl -s "$SERVICE_URL" > /dev/null 2>&1; then
    echo "❌ Service is not available at $SERVICE_URL"
    echo "Please start the service first:"
    echo "  docker-compose up -d monitor-service"
    echo "  # or"
    echo "  go run cmd/monitor-service/main.go"
    exit 1
fi
echo "✅ Service is available"
echo ""

# Helper function to run load test
run_load_test() {
    local name="$1"
    local method="$2"
    local url="$3"
    local body="$4"
    
    echo "Running: $name"
    echo "  Method: $method"
    echo "  URL: $url"
    
    if [ -n "$body" ]; then
        echo "POST $url" | vegeta attack -duration="$DURATION" -rate="$RATE" -payload="$body" -header="Content-Type: application/json" | vegeta report
    else
        echo "$method $url" | vegeta attack -duration="$DURATION" -rate="$RATE" | vegeta report
    fi
    
    echo ""
}

# Test 1: Create Maintenance Window
run_load_test "Create Maintenance Window" \
    "POST" \
    "$SERVICE_URL/api/v1/maintenance/windows" \
    '{
        "name": "Load Test Window",
        "start_time": "2026-03-28T02:00:00Z",
        "end_time": "2026-03-28T04:00:00Z",
        "recurrence": "RECURRENCE_TYPE_ONCE",
        "is_global": false,
        "pause_monitoring": true,
        "suppress_alerts": true,
        "safe_mode": false,
        "monitor_ids": []
    }'

# Test 2: List Maintenance Windows
run_load_test "List Maintenance Windows" \
    "GET" \
    "$SERVICE_URL/api/v1/maintenance/windows" \
    ""

# Test 3: Get Maintenance Window History
run_load_test "Get Maintenance Window History" \
    "GET" \
    "$SERVICE_URL/api/v1/maintenance/windows/history" \
    ""

echo "=== Load Testing Complete ==="
echo "Check vegeta reports above for:"
echo "  - Requests per second (RPS)"
echo "  - Latency (p50, p95, p99)"
echo "  - Error rate"
echo ""
echo "Targets:"
echo "  GET requests: < 100ms (p95)"
echo "  POST requests: < 200ms (p95)"
echo "  Error rate: < 1%"
