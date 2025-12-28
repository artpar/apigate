#!/bin/bash
# Quick UI test script - run after each code change

set -e

BASE="http://localhost:8080"

echo "=== APIGate UI Test ==="

# Rebuild
echo "Building..."
go build -o apigate ./cmd/apigate

# Restart server
echo "Restarting server..."
pkill -f "apigate serve" 2>/dev/null || true
sleep 1
rm -f data/test.db*
./apigate serve --config configs/test.yaml &
SERVER_PID=$!
sleep 2

# Test pages
echo ""
echo "Testing pages..."

test_page() {
    local path=$1
    local expected=$2
    if curl -s "$BASE$path" | grep -q "$expected"; then
        echo "✓ $path"
    else
        echo "✗ $path (expected: $expected)"
    fi
}

test_page "/login" "Admin Login"
test_page "/setup" "Welcome to APIGate"
test_page "/health" '{"status":"ok"}'

# Test setup flow
echo ""
echo "Testing setup flow..."
curl -s -X POST "$BASE/setup/step/1" \
    -d "admin_email=admin@test.com" \
    -d "admin_password=testpass123" \
    -d "admin_password_confirm=testpass123" \
    -c cookies.txt -b cookies.txt > /dev/null

# Test login
echo "Testing login..."
RESULT=$(curl -s -X POST "$BASE/login" \
    -d "email=admin@test.com" \
    -d "password=testpass123" \
    -c cookies.txt -b cookies.txt -w "%{http_code}" -o /dev/null)

if [ "$RESULT" = "302" ]; then
    echo "✓ Login works (redirected)"
else
    echo "✗ Login failed (status: $RESULT)"
fi

# Test authenticated pages
echo ""
echo "Testing authenticated pages..."
test_auth_page() {
    local path=$1
    local expected=$2
    if curl -s -b cookies.txt "$BASE$path" | grep -q "$expected"; then
        echo "✓ $path (authenticated)"
    else
        echo "✗ $path"
    fi
}

test_auth_page "/dashboard" "Dashboard"
test_auth_page "/users" "Users"
test_auth_page "/keys" "API Keys"
test_auth_page "/plans" "Plans"
test_auth_page "/settings" "Settings"

# Cleanup
rm -f cookies.txt
kill $SERVER_PID 2>/dev/null

echo ""
echo "=== Tests Complete ==="
