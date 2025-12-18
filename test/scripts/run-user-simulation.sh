#!/bin/sh
# Run user simulation for stress testing

set -e

# Wait for services
/test/scripts/wait-for-services.sh

echo "========================================"
echo "Running contextd User Simulation"
echo "========================================"
echo ""
echo "Configuration:"
echo "  SIM_DEVELOPERS: ${SIM_DEVELOPERS:-3}"
echo "  SIM_DURATION: ${SIM_DURATION:-60s}"
echo "  QDRANT_HOST: ${QDRANT_HOST:-localhost}"
echo "  QDRANT_PORT: ${QDRANT_PORT:-6334}"
echo ""

cd /contextd

# Run user simulation test
go test -v -count=1 -run "TestUserSimulation" \
    -timeout "${SIM_DURATION}" \
    ./test/integration/framework/... 2>&1 | tee /results/simulation-output.txt

EXIT_CODE=$?

echo ""
echo "========================================"
if [ $EXIT_CODE -eq 0 ]; then
    echo "User simulation completed successfully"
else
    echo "User simulation had errors"
fi
echo "========================================"

exit $EXIT_CODE
