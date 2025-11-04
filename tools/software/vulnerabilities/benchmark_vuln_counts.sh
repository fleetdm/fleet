#!/bin/bash

# Performance testing script for vulnerability count updates
# Usage: ./benchmark_vuln_counts.sh [small|medium|large|custom]

set -e

TOOL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/performance_test" && pwd)"
TOOL_PATH="$TOOL_DIR/test_vuln_counts_performance.go"

# Default MySQL settings (adjust as needed)
export MYSQL_TEST=1

run_test() {
    local name="$1"
    local hosts="$2"
    local teams="$3"
    local cves="$4"
    local iterations="$5"

    echo ""
    echo "=========================================="
    echo "Running $name test scenario"
    echo "Hosts: $hosts, Teams: $teams, CVEs: $cves"
    echo "=========================================="

    # Seed data
    echo "Seeding test data..."
    go run "$TOOL_PATH" \
        -hosts="$hosts" \
        -teams="$teams" \
        -cves="$cves" \
        -seed-only

    echo ""
    echo "Running performance test..."
    go run "$TOOL_PATH" \
        -iterations="$iterations" \
        -test-only
}

case "${1:-medium}" in
    "small")
        echo "Running SMALL test scenario"
        run_test "SMALL" 50 3 100 5
        ;;
    "medium")
        echo "Running MEDIUM test scenario"
        run_test "MEDIUM" 200 10 500 3
        ;;
    "large")
        echo "Running LARGE test scenario"
        run_test "LARGE" 1000 15 2000 3
        ;;
    "xlarge")
        echo "Running EXTRA LARGE test scenario"
        run_test "XLARGE" 5000 20 5000 2
        ;;
    "custom")
        # Allow custom parameters
        HOSTS=${2:-100}
        TEAMS=${3:-5}
        CVES=${4:-500}
        ITERATIONS=${5:-3}
        run_test "CUSTOM" "$HOSTS" "$TEAMS" "$CVES" "$ITERATIONS"
        ;;
    *)
        echo "Usage: $0 [small|medium|large|xlarge|custom]"
        echo ""
        echo "Predefined scenarios:"
        echo "  small:   50 hosts,   3 teams,   100 CVEs"
        echo "  medium:  200 hosts,  10 teams,  500 CVEs"
        echo "  large:   1000 hosts, 25 teams,  2000 CVEs"
        echo "  xlarge:  5000 hosts, 50 teams,  5000 CVEs"
        echo ""
        echo "Custom usage: $0 custom <hosts> <teams> <cves> [iterations]"
        echo "Example: $0 custom 300 15 800 5"
        exit 1
        ;;
esac

echo ""
echo "Performance test complete!"
echo "You can now test your optimized implementation and compare results."