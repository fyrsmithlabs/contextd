#!/bin/sh
# Run e2e tests in container

set -e

# Wait for services
/test/scripts/wait-for-services.sh

echo "========================================"
echo "Running contextd E2E Tests"
echo "========================================"
echo ""
echo "Environment:"
echo "  QDRANT_HOST: ${QDRANT_HOST:-localhost}"
echo "  QDRANT_PORT: ${QDRANT_PORT:-6334}"
echo "  TEST_SUITES: ${TEST_SUITES:-all}"
echo "  VERBOSE: ${VERBOSE:-true}"
echo ""

cd /contextd

# Set test flags
TEST_FLAGS="-v -count=1"
if [ "$VERBOSE" = "true" ]; then
    TEST_FLAGS="$TEST_FLAGS -v"
fi

# Run tests based on TEST_SUITES
case "$TEST_SUITES" in
    all)
        echo "Running all integration test suites..."
        go test $TEST_FLAGS ./test/integration/framework/... 2>&1 | tee /results/test-output.txt
        ;;
    policy)
        echo "Running Suite A - Policy compliance tests..."
        go test $TEST_FLAGS -run "TestSuiteA" ./test/integration/framework/... 2>&1 | tee /results/test-output.txt
        ;;
    bugfix)
        echo "Running Suite C - Bug-fix learning tests..."
        go test $TEST_FLAGS -run "TestSuiteC" ./test/integration/framework/... 2>&1 | tee /results/test-output.txt
        ;;
    multisession)
        echo "Running Suite D - Multi-session tests..."
        go test $TEST_FLAGS -run "TestSuiteD" ./test/integration/framework/... 2>&1 | tee /results/test-output.txt
        ;;
    secrets)
        echo "Running Suite A - Secrets tests..."
        go test $TEST_FLAGS -run "TestSuiteA_Secrets" ./test/integration/framework/... 2>&1 | tee /results/test-output.txt
        ;;
    workflow)
        echo "Running Temporal workflow tests..."
        go test $TEST_FLAGS -run "TestWorkflow" ./test/integration/framework/... 2>&1 | tee /results/test-output.txt
        ;;
    *)
        echo "Unknown test suite: $TEST_SUITES"
        echo "Available: all, policy, bugfix, multisession, secrets, workflow"
        exit 1
        ;;
esac

EXIT_CODE=$?

echo ""
echo "========================================"
if [ $EXIT_CODE -eq 0 ]; then
    echo "All tests PASSED"
else
    echo "Some tests FAILED"
fi
echo "========================================"
echo ""
echo "Test output saved to /results/test-output.txt"

exit $EXIT_CODE
