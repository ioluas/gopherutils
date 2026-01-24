#!/usr/bin/env bash

# Pre-push hook
# Runs CQ target and fails if coverage is below 80% for any file

if [ "$SKIP_PREPUSH" = "1" ]; then
    echo "SKIP_PREPUSH is set to 1, skipping hook execution."
    exit 0
fi

check_coverage() {
    # Calculates per-file coverage from coverage.txt and fails if any file is below 80%
    COVERAGE_FILE="coverage.txt"
    THRESHOLD=80

    if [ ! -f "$COVERAGE_FILE" ]; then
        echo "Error: $COVERAGE_FILE not found. Run 'make coverage' first."
        return 1
    fi

    # Get list of unique files in the coverage report
    FILES=$(grep -v "^mode:" "$COVERAGE_FILE" | cut -d: -f1 | sort | uniq)

    FAILED=0

    for FILE in $FILES; do
        # For each file, we sum the statements and the covered statements.
        # coverage.txt format: name.go:line.col,line.col num_statements count

        # Total statements: sum of the second to last field for this file
        TOTAL_STMTS=$(awk -v f="$FILE:" 'index($0, f) == 1 {sum += $2} END {print sum}' "$COVERAGE_FILE")

        # Covered statements: sum of the second to last field for this file where the last field (count) > 0
        COVERED_STMTS=$(awk -v f="$FILE:" 'index($0, f) == 1 && $3 > 0 {sum += $2} END {print sum}' "$COVERAGE_FILE")

        # Handle case where TOTAL_STMTS is 0 (should not happen for files in coverage.txt)
        if [ "$TOTAL_STMTS" -eq 0 ]; then
            PERCENTAGE=100
        else
            # Calculate percentage using integer arithmetic
            PERCENTAGE=$(( COVERED_STMTS * 100 / TOTAL_STMTS ))
        fi

        # Display result
        echo "File: $FILE - Coverage: $PERCENTAGE% ($COVERED_STMTS/$TOTAL_STMTS statements)"

        # Check against threshold
        if [ "$PERCENTAGE" -lt "$THRESHOLD" ]; then
            echo "  FAILED: Coverage is below $THRESHOLD%"
            FAILED=1
        fi
    done

    if [ "$FAILED" -eq 1 ]; then
        echo "Coverage check failed!"
        return 1
    else
        echo "Coverage check passed!"
        return 0
    fi
}

echo "Running pre-push hook..."

# Run CQ target (lint, fmt, coverage)
if ! make CQ; then
    echo "make CQ failed. Push aborted."
    exit 1
fi

# Run coverage check function
if ! check_coverage; then
    echo "Coverage check failed. Push aborted."
    exit 1
fi

echo "Pre-push hook passed."
exit 0
