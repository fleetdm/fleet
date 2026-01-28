#!/bin/bash

# Demo script for the gm label-history command
# This script demonstrates various use cases of the label-history feature

set -e

echo "=== GitHub Label History Tool Demo ==="
echo ""
echo "This demo showcases the label-history command which finds issues"
echo "that have ever had a specific label applied to them."
echo ""

# Check if gm is built
if [ ! -f "./gm" ]; then
    echo "Building gm tool..."
    go build -o gm cmd/gm/*.go
    echo "Build complete."
    echo ""
fi

echo "=== Example 1: Find all issues ever labeled as 'bug' ==="
echo "Command: ./gm label-history fleetdm/fleet --start-date 2024-06-01 --label 'bug'"
echo ""
./gm label-history fleetdm/fleet --start-date 2024-06-01 --label "bug"
echo ""

echo "=== Example 2: Same query with JSON output ==="
echo "Command: ./gm label-history fleetdm/fleet --start-date 2024-06-01 --label 'bug' --json"
echo ""
./gm label-history fleetdm/fleet --start-date 2024-06-01 --label "bug" --json
echo ""

echo "=== Example 3: Find issues with a specific team label ==="
echo "Command: ./gm label-history fleetdm/fleet --start-date 2024-01-01 --label '#g-software'"
echo ""
./gm label-history fleetdm/fleet --start-date 2024-01-01 --label "#g-software"
echo ""

echo "=== Example 4: Search for issues with special character labels ==="
echo "Command: ./gm label-history fleetdm/fleet --start-date 2024-01-01 --label ':product'"
echo ""
./gm label-history fleetdm/fleet --start-date 2024-01-01 --label ":product"
echo ""

echo "=== Demo Complete ==="
echo ""
echo "The label-history command supports:"
echo "  - Any public or private GitHub repository"
echo "  - Historical label tracking (labels that were removed)"
echo "  - Date filtering to limit search scope"
echo "  - Both human-readable and JSON output"
echo "  - Labels with special characters"
echo ""
echo "For more information, see README.md or run: ./gm label-history --help"
