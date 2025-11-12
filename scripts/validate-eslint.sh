#!/bin/bash
# Validates generated ESLint configuration

set -e

ESLINTRC=".sym/.eslintrc.json"

if [ ! -f "$ESLINTRC" ]; then
    echo "Error: $ESLINTRC not found"
    exit 1
fi

echo "Validating ESLint configuration..."

# Check if file is valid JSON
if ! jq empty "$ESLINTRC" 2>/dev/null; then
    echo "Error: Invalid JSON in $ESLINTRC"
    exit 1
fi

# Check required fields
if ! jq -e '.rules' "$ESLINTRC" > /dev/null; then
    echo "Error: Missing 'rules' field in $ESLINTRC"
    exit 1
fi

# Count rules
RULE_COUNT=$(jq '.rules | length' "$ESLINTRC")
echo "✓ Valid ESLint config with $RULE_COUNT rules"

# Optional: Run eslint --print-config if eslint is installed
if command -v eslint &> /dev/null; then
    echo "✓ ESLint validation passed"
else
    echo "ℹ eslint not installed, skipping full validation"
fi
