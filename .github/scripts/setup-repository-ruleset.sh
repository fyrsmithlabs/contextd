#!/bin/bash
# Setup Repository Ruleset via GitHub API
# Uses the newer Rulesets API instead of legacy branch protection
# Requires: gh CLI tool or GITHUB_TOKEN environment variable

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REPO_OWNER="${1:-dahendel}"
REPO_NAME="${2:-contextd}"

echo -e "${BLUE}╔════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║${NC}  GitHub Repository Ruleset Setup                       ${BLUE}║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}Repository:${NC} ${REPO_OWNER}/${REPO_NAME}"
echo ""

# Check if gh CLI is available
if command -v gh &> /dev/null; then
    echo -e "${GREEN}✓ GitHub CLI found${NC}"
    USE_GH_CLI=true
elif [ -n "$GITHUB_TOKEN" ]; then
    echo -e "${GREEN}✓ GITHUB_TOKEN found${NC}"
    USE_GH_CLI=false
else
    echo -e "${RED}✗ Neither gh CLI nor GITHUB_TOKEN found${NC}"
    echo ""
    echo "Please either:"
    echo "1. Install GitHub CLI: https://cli.github.com/"
    echo "2. Set GITHUB_TOKEN environment variable"
    echo ""
    exit 1
fi

# Function to make GitHub API call
github_api() {
    local method="$1"
    local endpoint="$2"
    local data="$3"

    if [ "$USE_GH_CLI" = true ]; then
        if [ -n "$data" ]; then
            echo "$data" | gh api -X "$method" "$endpoint" --input -
        else
            gh api -X "$method" "$endpoint"
        fi
    else
        if [ -n "$data" ]; then
            curl -s -X "$method" \
                -H "Authorization: token $GITHUB_TOKEN" \
                -H "Accept: application/vnd.github+json" \
                -H "X-GitHub-Api-Version: 2022-11-28" \
                "https://api.github.com$endpoint" \
                -d "$data"
        else
            curl -s -X "$method" \
                -H "Authorization: token $GITHUB_TOKEN" \
                -H "Accept: application/vnd.github+json" \
                -H "X-GitHub-Api-Version: 2022-11-28" \
                "https://api.github.com$endpoint"
        fi
    fi
}

echo -e "${BLUE}Creating repository ruleset...${NC}"
echo ""

# Repository ruleset configuration
# Targets: main and develop branches
RULESET_CONFIG=$(cat <<'EOF'
{
  "name": "Main Branch Protection",
  "target": "branch",
  "enforcement": "active",
  "bypass_actors": [],
  "conditions": {
    "ref_name": {
      "include": [
        "refs/heads/main",
        "refs/heads/develop"
      ],
      "exclude": []
    }
  },
  "rules": [
    {
      "type": "pull_request",
      "parameters": {
        "required_approving_review_count": 1,
        "dismiss_stale_reviews_on_push": true,
        "require_code_owner_review": true,
        "require_last_push_approval": false,
        "required_review_thread_resolution": true
      }
    },
    {
      "type": "required_status_checks",
      "parameters": {
        "strict_required_status_checks_policy": true,
        "required_status_checks": [
          {
            "context": "Documentation Validation",
            "integration_id": null
          },
          {
            "context": "Markdown Linting",
            "integration_id": null
          },
          {
            "context": "Link Validation",
            "integration_id": null
          },
          {
            "context": "Repository Size Check",
            "integration_id": null
          }
        ]
      }
    },
    {
      "type": "deletion"
    },
    {
      "type": "non_fast_forward"
    },
    {
      "type": "required_linear_history"
    }
  ]
}
EOF
)

echo -e "${YELLOW}Creating ruleset for main and develop branches...${NC}"
RESPONSE=$(github_api POST "/repos/${REPO_OWNER}/${REPO_NAME}/rulesets" "$RULESET_CONFIG")

if echo "$RESPONSE" | grep -q '"id"'; then
    RULESET_ID=$(echo "$RESPONSE" | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
    echo -e "${GREEN}✓ Repository ruleset created successfully (ID: ${RULESET_ID})${NC}"
else
    echo -e "${RED}✗ Failed to create repository ruleset${NC}"
    echo "Response: $RESPONSE"
    exit 1
fi

echo ""
echo -e "${BLUE}Configuration Summary:${NC}"
echo ""
echo "  ${GREEN}✓${NC} Ruleset Name: Main Branch Protection"
echo "  ${GREEN}✓${NC} Enforcement: Active"
echo "  ${GREEN}✓${NC} Applies to: main, develop branches"
echo ""
echo "  ${BLUE}Pull Request Rules:${NC}"
echo "    • Require 1 approving review"
echo "    • Dismiss stale reviews on push"
echo "    • Require code owner review"
echo "    • Require conversation resolution"
echo ""
echo "  ${BLUE}Status Check Rules:${NC}"
echo "    • Require branches to be up to date"
echo "    • Required checks:"
echo "      - Documentation Validation"
echo "      - Markdown Linting"
echo "      - Link Validation"
echo "      - Repository Size Check"
echo ""
echo "  ${BLUE}Protection Rules:${NC}"
echo "    • Block force pushes (non-fast-forward)"
echo "    • Block branch deletion"
echo "    • Require linear history"
echo ""

echo -e "${BLUE}Additional Recommendations:${NC}"
echo ""
echo "1. ${YELLOW}Verify CODEOWNERS file${NC}"
echo "   Location: .github/CODEOWNERS"
echo "   Current owner: @${REPO_OWNER}"
echo ""
echo "2. ${YELLOW}Enable Security Features${NC}"
echo "   Go to: Settings > Code security and analysis"
echo "   - Enable Dependabot alerts"
echo "   - Enable Dependabot security updates"
echo "   - Enable Secret scanning"
echo ""
echo "3. ${YELLOW}Consider Additional Rulesets${NC}"
echo "   - Feature branch naming conventions"
echo "   - Tag protection rules"
echo "   - Required workflows"
echo ""
echo "4. ${YELLOW}Setup Bypass Actors (if needed)${NC}"
echo "   Go to: Settings > Rules > Rulesets"
echo "   Add service accounts or automation users"
echo ""

echo -e "${GREEN}Repository ruleset setup complete!${NC}"
echo ""
echo "View and manage your rulesets at:"
echo "https://github.com/${REPO_OWNER}/${REPO_NAME}/settings/rules"
echo ""
echo "Learn more about rulesets:"
echo "https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets"
