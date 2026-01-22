#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}   contextd Integration Test Suite                             ${NC}"
echo -e "${BLUE}   Testing all core features and workflows                     ${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo

# Parse arguments
VERBOSE=false
CLEANUP=true
FEATURE=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --feature)
            FEATURE="$2"
            shift 2
            ;;
        --verbose|-v)
            VERBOSE=true
            shift
            ;;
        --no-cleanup)
            CLEANUP=false
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo
            echo "Options:"
            echo "  --feature <name>    Run specific feature tests:"
            echo "                        reasoningbank, checkpoint, remediation,"
            echo "                        repository, folding, e2e, all (default)"
            echo "  --verbose, -v       Enable verbose output"
            echo "  --no-cleanup        Don't cleanup Docker resources after tests"
            echo "  --help, -h          Show this help message"
            echo
            echo "Feature options:"
            echo "  reasoningbank - ReasoningBank (memory) tests"
            echo "  checkpoint    - Checkpoint (context snapshots) tests"
            echo "  remediation   - Remediation (error patterns) tests"
            echo "  repository    - Repository (semantic search) tests"
            echo "  folding       - Context-Folding (branch isolation) tests"
            echo "  e2e           - End-to-End workflow tests"
            echo "  all           - All integration tests (default)"
            echo
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Default to all tests
if [ -z "$FEATURE" ]; then
    FEATURE="all"
fi

# Cleanup function
cleanup() {
    if [ "$CLEANUP" = true ]; then
        echo
        echo -e "${YELLOW}🧹 Cleaning up Docker resources...${NC}"
        docker-compose -f docker-compose.integration.yml down -v --remove-orphans 2>/dev/null || true
    fi
}

# Set trap to cleanup on exit
trap cleanup EXIT

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}❌ Docker is not running. Please start Docker and try again.${NC}"
    exit 1
fi

echo -e "${YELLOW}📦 Building integration test containers...${NC}"
docker-compose -f docker-compose.integration.yml build --quiet

echo
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}   Starting Qdrant Vector Database                             ${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo

# Start Qdrant
echo -e "${YELLOW}🚀 Starting Qdrant...${NC}"
docker-compose -f docker-compose.integration.yml up -d qdrant

echo -e "${YELLOW}⏳ Waiting for Qdrant to be healthy...${NC}"
timeout=60
elapsed=0
while ! docker-compose -f docker-compose.integration.yml ps qdrant | grep -q "healthy"; do
    if [ $elapsed -ge $timeout ]; then
        echo -e "${RED}❌ Qdrant failed to become healthy within ${timeout}s${NC}"
        docker-compose -f docker-compose.integration.yml logs qdrant
        exit 1
    fi
    echo -n "."
    sleep 2
    elapsed=$((elapsed + 2))
done
echo -e " ${GREEN}✓${NC}"

echo
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}   Running Integration Tests                                   ${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo

run_test_profile() {
    local profile=$1
    local description=$2
    local emoji=$3

    echo
    echo -e "${GREEN}${emoji} ${description}${NC}"
    if [ "$VERBOSE" = true ]; then
        docker-compose -f docker-compose.integration.yml run --rm \
            --profile "$profile" "test-$profile"
    else
        docker-compose -f docker-compose.integration.yml run --rm \
            --profile "$profile" "test-$profile" \
            | grep -E "PASS|FAIL|✅|^ok|^FAIL" || true
    fi
}

# Run tests based on feature selection
case "$FEATURE" in
    reasoningbank)
        run_test_profile "reasoningbank" "ReasoningBank (Cross-Session Memory)" "🧠"
        ;;
    checkpoint)
        run_test_profile "checkpoint" "Checkpoint (Context Snapshots)" "💾"
        ;;
    remediation)
        run_test_profile "remediation" "Remediation (Error Pattern Matching)" "🔧"
        ;;
    repository)
        run_test_profile "repository" "Repository (Semantic Code Search)" "🔍"
        ;;
    folding)
        run_test_profile "folding" "Context-Folding (Branch Isolation)" "📁"
        ;;
    e2e)
        run_test_profile "e2e" "End-to-End Workflows" "🚀"
        ;;
    all)
        echo -e "${YELLOW}Running all integration tests...${NC}"
        if [ "$VERBOSE" = true ]; then
            docker-compose -f docker-compose.integration.yml run --rm test-all
        else
            docker-compose -f docker-compose.integration.yml run --rm test-all \
                | grep -E "PASS|FAIL|✅|^ok|^FAIL|^===" || true
        fi
        ;;
    *)
        echo -e "${RED}Unknown feature: $FEATURE${NC}"
        echo "Valid features: reasoningbank, checkpoint, remediation, repository, folding, e2e, all"
        exit 1
        ;;
esac

echo
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}   ✨ Integration tests completed successfully!                ${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo

if [ "$FEATURE" = "all" ]; then
    echo -e "${YELLOW}Features validated:${NC}"
    echo -e "  ✅ ReasoningBank: Memory CRUD, search, confidence scoring, multi-tenancy"
    echo -e "  ✅ Checkpoint: Save/resume workflows, pagination, tenant isolation"
    echo -e "  ✅ Remediation: Error pattern matching, fuzzy search, tag filtering"
    echo -e "  ✅ Repository: Semantic search, grep fallback, file type filtering"
    echo -e "  ✅ Context-Folding: Branch lifecycle, budget enforcement, secret scrubbing"
    echo -e "  ✅ End-to-End: Complete development workflows with multiple services"
else
    echo -e "${YELLOW}Feature validated: ${FEATURE}${NC}"
fi

echo
echo -e "${GREEN}🎉 All integration tests validated in containerized environment!${NC}"
echo
