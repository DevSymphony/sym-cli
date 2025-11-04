#!/bin/bash
set -e

# Symphony MCP Server E2E Test
# Tests both stdio and HTTP modes

echo "=========================================="
echo "Symphony MCP Server - E2E Tests"
echo "=========================================="

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Function to print test result
print_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✅ $2${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}❌ $2${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# Setup: Create temporary directory with test policy
TEST_DIR=$(mktemp -d)
trap "rm -rf $TEST_DIR" EXIT

echo ""
echo "Setup: Creating test environment..."
echo "Test directory: $TEST_DIR"

# Create test policy file
mkdir -p "$TEST_DIR/.sym"
cat > "$TEST_DIR/.sym/user-policy.json" <<'EOF'
{
  "version": "1.0.0",
  "defaults": {
    "languages": ["go", "typescript"],
    "severity": "warning"
  },
  "rules": [
    {
      "say": "Functions should be documented",
      "category": "documentation"
    },
    {
      "say": "Lines should be less than 100 characters",
      "category": "formatting",
      "params": { "max": 100 }
    },
    {
      "say": "No hardcoded secrets",
      "category": "security",
      "severity": "error"
    }
  ]
}
EOF

echo "✅ Test policy created"

# Determine how to run sym command
if [ -n "$SYM_BINARY" ] && [ -f "$SYM_BINARY" ]; then
    SYM_CMD="$SYM_BINARY"
    echo "Using SYM_BINARY: $SYM_CMD"
elif command -v sym &> /dev/null; then
    SYM_CMD="sym"
    echo "Using installed 'sym' command"
elif [ -f "$PROJECT_ROOT/bin/sym" ]; then
    SYM_CMD="$PROJECT_ROOT/bin/sym"
    echo "Using local binary: $SYM_CMD"
elif [ -f "$PROJECT_ROOT/bin/sym-linux-amd64" ]; then
    SYM_CMD="$PROJECT_ROOT/bin/sym-linux-amd64"
    echo "Using local binary: $SYM_CMD"
elif command -v npx &> /dev/null; then
    SYM_CMD="npx @devsymphony/sym"
    echo "Using npx @devsymphony/sym"
else
    echo -e "${RED}❌ No sym command found. Please build or install sym first.${NC}"
    exit 1
fi

# Test 1: Help command
echo ""
echo "Test 1: MCP help command"
echo "------------------------"
if $SYM_CMD mcp --help &> /dev/null; then
    print_result 0 "MCP help command works"
else
    print_result 1 "MCP help command failed"
fi

# Test 2: stdio mode - JSON-RPC request
echo ""
echo "Test 2: stdio mode - query_conventions"
echo "---------------------------------------"
REQUEST='{"jsonrpc":"2.0","method":"query_conventions","params":{},"id":1}'
RESPONSE=$(echo "$REQUEST" | timeout 5 $SYM_CMD mcp --config "$TEST_DIR/.sym/user-policy.json" 2>/dev/null || echo "TIMEOUT")

if echo "$RESPONSE" | grep -q '"result"'; then
    print_result 0 "stdio mode query_conventions works"
    echo "Response preview: $(echo $RESPONSE | head -c 100)..."
else
    print_result 1 "stdio mode query_conventions failed"
    echo "Response: $RESPONSE"
fi

# Test 3: stdio mode - validate_code
echo ""
echo "Test 3: stdio mode - validate_code"
echo "-----------------------------------"
# Create a test file
cat > "$TEST_DIR/test.go" <<'EOF'
package main

func main() {
    println("Hello")
}
EOF

REQUEST='{"jsonrpc":"2.0","method":"validate_code","params":{"files":["'$TEST_DIR'/test.go"]},"id":2}'
RESPONSE=$(echo "$REQUEST" | timeout 5 $SYM_CMD mcp --config "$TEST_DIR/.sym/user-policy.json" 2>/dev/null || echo "TIMEOUT")

if echo "$RESPONSE" | grep -q '"result"'; then
    print_result 0 "stdio mode validate_code works"
else
    print_result 1 "stdio mode validate_code failed"
    echo "Response: $RESPONSE"
fi

# Test 4: HTTP mode - start server
echo ""
echo "Test 4: HTTP mode - server start"
echo "---------------------------------"
HTTP_PORT=14000  # Use non-standard port to avoid conflicts

# Start HTTP server in background
$SYM_CMD mcp --config "$TEST_DIR/.sym/user-policy.json" --port $HTTP_PORT &> "$TEST_DIR/mcp-server.log" &
MCP_PID=$!

# Wait for server to start
sleep 2

# Check if server is running
if kill -0 $MCP_PID 2>/dev/null; then
    print_result 0 "HTTP mode server started (PID: $MCP_PID)"
else
    print_result 1 "HTTP mode server failed to start"
    cat "$TEST_DIR/mcp-server.log"
fi

# Test 5: HTTP mode - health check
echo ""
echo "Test 5: HTTP mode - health check"
echo "---------------------------------"
if command -v curl &> /dev/null; then
    HEALTH_RESPONSE=$(curl -s http://localhost:$HTTP_PORT/health 2>/dev/null || echo "FAILED")

    if echo "$HEALTH_RESPONSE" | grep -q '"status"'; then
        print_result 0 "HTTP health check passed"
        echo "Response: $HEALTH_RESPONSE"
    else
        print_result 1 "HTTP health check failed"
        echo "Response: $HEALTH_RESPONSE"
    fi
else
    echo -e "${YELLOW}⚠️  curl not available, skipping HTTP tests${NC}"
fi

# Test 6: HTTP mode - query_conventions
echo ""
echo "Test 6: HTTP mode - query_conventions"
echo "--------------------------------------"
if command -v curl &> /dev/null; then
    RPC_REQUEST='{"jsonrpc":"2.0","method":"query_conventions","params":{"category":"naming"},"id":1}'
    RPC_RESPONSE=$(curl -s -X POST http://localhost:$HTTP_PORT \
        -H "Content-Type: application/json" \
        -d "$RPC_REQUEST" 2>/dev/null || echo "FAILED")

    if echo "$RPC_RESPONSE" | grep -q '"result"'; then
        print_result 0 "HTTP query_conventions passed"
    else
        print_result 1 "HTTP query_conventions failed"
        echo "Response: $RPC_RESPONSE"
    fi
else
    echo -e "${YELLOW}⚠️  curl not available, skipping${NC}"
fi

# Cleanup: Stop HTTP server
if kill -0 $MCP_PID 2>/dev/null; then
    echo ""
    echo "Cleanup: Stopping MCP server (PID: $MCP_PID)..."
    kill $MCP_PID
    wait $MCP_PID 2>/dev/null || true
    echo "✅ Server stopped"
fi

# Print summary
echo ""
echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
if [ $TESTS_FAILED -gt 0 ]; then
    echo -e "${RED}Failed: $TESTS_FAILED${NC}"
else
    echo -e "Failed: $TESTS_FAILED"
fi
echo "=========================================="

# Exit with appropriate code
if [ $TESTS_FAILED -gt 0 ]; then
    exit 1
else
    exit 0
fi
