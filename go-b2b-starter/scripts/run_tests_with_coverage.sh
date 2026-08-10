#!/bin/bash

# File: scripts/run_tests_with_coverage.sh

set -o pipefail

echo "Running tests with coverage for the root module..."

# Create coverage directory
mkdir -p coverage
rm -f coverage/coverage.txt coverage/coverage.out

# Run tests for the single root Go module
if ! go test -v -coverprofile=coverage/coverage.out ./...; then
    echo "Tests FAILED for module root" >&2
    exit 1
fi

if [ -s coverage/coverage.out ]; then
    echo "mode: atomic" > coverage/coverage.txt
    tail -n +2 coverage/coverage.out >> coverage/coverage.txt
else
    echo "No coverage data generated" >&2
    exit 1
fi

rm -f coverage/coverage.out

# Remove any non-coverage lines (like file headers)
sed -i '/^[^[:space:]]*:/!d' coverage/coverage.txt

# Generate coverage reports
if [ -s coverage/coverage.txt ]; then
    go tool cover -func=coverage/coverage.txt
    go tool cover -html=coverage/coverage.txt -o coverage/coverage.html
    echo "Coverage report generated in coverage/coverage.html"
else
    echo "No coverage data generated"
    exit 1
fi
