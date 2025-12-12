#!/usr/bin/env bash
#
# contextd Docker Setup - Deprecated
#
# The Docker variant has been removed from the plugin.
# See docs/DOCKER.md for manual Docker setup instructions.
#

echo "========================================="
echo "  Docker variant removed"
echo "========================================="
echo
echo "The contextd Docker plugin variant has been removed."
echo "The plugin now installs the native binary only."
echo
echo "For Docker usage, see the documentation:"
echo "  https://github.com/fyrsmithlabs/contextd/blob/main/docs/DOCKER.md"
echo
echo "To install the native binary instead, run:"
echo "  /plugin install contextd@fyrsmithlabs/contextd"
echo
exit 1
