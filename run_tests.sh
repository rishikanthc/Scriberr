#!/bin/bash

echo "🧪 Running Scriberr Backend Unit Tests"
echo "======================================"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to run tests and capture results
run_test() {
    local test_name=$1
    local test_files=$2
    
    echo -e "\n${YELLOW}🔄 Running $test_name...${NC}"
    
    if go test $test_files -v; then
        echo -e "${GREEN}✅ $test_name PASSED${NC}"
        return 0
    else
        echo -e "${RED}❌ $test_name FAILED${NC}"
        return 1
    fi
}

# Track results
passed=0
failed=0
total=0

# Run individual test suites
echo -e "\n${YELLOW}Running individual test suites:${NC}"

# Security Tests (known working)
if run_test "Security Tests" "./tests/security_test.go"; then
    ((passed++))
else
    ((failed++))
fi
((total++))

# Auth Service Tests (known working)
if run_test "Authentication Service Tests" "./tests/test_helpers.go ./tests/auth_service_test.go"; then
    ((passed++))
else
    ((failed++))
fi
((total++))

# LLM Tests (known working)
if run_test "LLM Integration Tests" "./tests/test_helpers.go ./tests/llm_test.go"; then
    ((passed++))
else
    ((failed++))
fi
((total++))

# Database Tests (may have issues)
if run_test "Database Tests" "./tests/test_helpers.go ./tests/database_test.go"; then
    ((passed++))
else
    ((failed++))
fi
((total++))

# Queue Tests (may have issues)
if run_test "Queue Management Tests" "./tests/test_helpers.go ./tests/queue_test.go"; then
    ((passed++))
else
    ((failed++))
fi
((total++))

# API Handler Tests (may have issues)
if run_test "API Handler Tests" "./tests/test_helpers.go ./tests/api_handlers_test.go"; then
    ((passed++))
else
    ((failed++))
fi
((total++))

# Transcription Tests (may have issues)
if run_test "Transcription Service Tests" "./tests/test_helpers.go ./tests/transcription_service_test.go"; then
    ((passed++))
else
    ((failed++))
fi
((total++))

# Final summary
echo -e "\n======================================"
echo -e "${YELLOW}📊 TEST SUMMARY${NC}"
echo -e "======================================"
echo -e "Total Test Suites: $total"
echo -e "${GREEN}✅ Passed: $passed${NC}"
echo -e "${RED}❌ Failed: $failed${NC}"

if [ $failed -eq 0 ]; then
    echo -e "\n${GREEN}🎉 ALL TESTS PASSED!${NC}"
    exit 0
else
    echo -e "\n${RED}⚠️  Some tests failed. Check output above for details.${NC}"
    exit 1
fi