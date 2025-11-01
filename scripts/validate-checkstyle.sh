#!/bin/bash
# Validates generated Checkstyle configuration

set -e

CHECKSTYLE_XML=".sym/checkstyle.xml"

if [ ! -f "$CHECKSTYLE_XML" ]; then
    echo "Error: $CHECKSTYLE_XML not found"
    exit 1
fi

echo "Validating Checkstyle configuration..."

# Check if file is valid XML
if ! xmllint --noout "$CHECKSTYLE_XML" 2>/dev/null; then
    echo "Error: Invalid XML in $CHECKSTYLE_XML"
    exit 1
fi

# Check required structure
if ! xmllint --xpath "//module[@name='Checker']" "$CHECKSTYLE_XML" > /dev/null 2>&1; then
    echo "Error: Missing Checker module in $CHECKSTYLE_XML"
    exit 1
fi

# Count modules
MODULE_COUNT=$(xmllint --xpath "count(//module[@name='TreeWalker']/module)" "$CHECKSTYLE_XML")
echo "✓ Valid Checkstyle config with $MODULE_COUNT modules"
