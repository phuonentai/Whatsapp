#!/bin/bash

# File: scripts/run_tests_with_coverage.sh

echo "Running tests with coverage for all modules..."

# Create coverage directory
mkdir -p coverage
rm -f coverage/coverage.txt

# Find all go.mod files and run tests
find ./src -name go.mod | while read -r mod_file; do
    mod_dir=$(dirname "$mod_file")
    mod_name=$(basename "$mod_dir")
    
    echo "Testing module: $mod_name"
    
    (
        cd "$mod_dir"
        if go test -v -coverprofile=coverage.out ./...; then
            if [ -s coverage.out ]; then
                echo "mode: atomic" > "../../coverage/coverage.$mod_name.txt"
                tail -n +2 coverage.out >> "../../coverage/coverage.$mod_name.txt"
            else
                echo "No coverage data generated for $mod_name"
            fi
        else
            echo "Tests failed for $mod_name"
        fi
        rm -f coverage.out
    )
done

# Combine all coverage files
echo "mode: atomic" > coverage/coverage.txt
find coverage -name 'coverage.*.txt' -print0 | xargs -0 tail -q -n +2 >> coverage/coverage.txt

# Remove any non-coverage lines (like file headers)
sed -i '/^[^[:space:]]*:/!d' coverage/coverage.txt

# Generate coverage reports
if [ -s coverage/coverage.txt ]; then
    go tool cover -func=coverage/coverage.txt
    go tool cover -html=coverage/coverage.txt -o coverage/coverage.html
    echo "Coverage report generated in coverage/coverage.html"
else
    echo "No coverage data generated"
fi