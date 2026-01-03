#!/usr/bin/env bash
# Setup pre-commit hooks for contextd

set -euo pipefail

echo "Setting up pre-commit hooks..."

# Check if pre-commit is installed
if ! command -v pre-commit &> /dev/null; then
    echo "Installing pre-commit..."

    # Try pip first
    if command -v pip3 &> /dev/null; then
        pip3 install pre-commit
    elif command -v pip &> /dev/null; then
        pip install pre-commit
    elif command -v brew &> /dev/null; then
        brew install pre-commit
    else
        echo "Error: Could not find pip or brew to install pre-commit"
        echo "Please install pre-commit manually: https://pre-commit.com/#install"
        exit 1
    fi
fi

# Install the hooks
echo "Installing hooks..."
pre-commit install

# Optionally run against all files
read -p "Run pre-commit against all files now? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    pre-commit run --all-files
fi

echo ""
echo "✓ Pre-commit hooks installed!"
echo ""
echo "The following checks will run on each commit:"
echo "  - go fmt, go vet, go imports"
echo "  - golangci-lint"
echo "  - go test -short"
echo "  - shellcheck (bash scripts)"
echo "  - gitleaks (secret scanning)"
echo "  - trailing whitespace, YAML validation, etc."
echo ""
echo "To run manually: pre-commit run --all-files"
echo "To skip hooks: git commit --no-verify"
