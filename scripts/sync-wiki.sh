#!/bin/bash
# sync-wiki.sh - Syncs docs/spec to the GitHub wiki
#
# Usage: ./scripts/sync-wiki.sh [--dry-run]
#
# The wiki must be initialized first via GitHub UI.
# This script is also run automatically by CI on push to main.

set -e

DRY_RUN=false
if [ "$1" = "--dry-run" ]; then
    DRY_RUN=true
    echo "DRY RUN - No changes will be pushed"
fi

REPO_ROOT=$(git rev-parse --show-toplevel)
WIKI_DIR="/tmp/apigate-wiki"

# Clone or update wiki
if [ -d "$WIKI_DIR" ]; then
    echo "Updating existing wiki clone..."
    cd "$WIKI_DIR"
    git pull --rebase
else
    echo "Cloning wiki..."
    git clone git@github.com:artpar/apigate.wiki.git "$WIKI_DIR"
    cd "$WIKI_DIR"
fi

# Sync spec files
echo "Syncing spec files..."
cp "$REPO_ROOT/docs/spec/README.md" "$WIKI_DIR/Home.md"
cp "$REPO_ROOT/docs/spec/json-api.md" "$WIKI_DIR/JSON-API-Format.md"
cp "$REPO_ROOT/docs/spec/error-codes.md" "$WIKI_DIR/Error-Codes.md"
cp "$REPO_ROOT/docs/spec/pagination.md" "$WIKI_DIR/Pagination.md"
cp "$REPO_ROOT/docs/spec/resource-types.md" "$WIKI_DIR/Resource-Types.md"
cp "$REPO_ROOT/docs/spec/tls-certificates.md" "$WIKI_DIR/TLS-Certificates.md"
cp "$REPO_ROOT/docs/spec/metering-api.md" "$WIKI_DIR/Metering-API.md"

# Show diff
echo ""
echo "Changes to sync:"
echo "----------------------------------------"
git diff --stat || echo "(no changes)"
echo "----------------------------------------"

if $DRY_RUN; then
    echo ""
    echo "DRY RUN complete. Run without --dry-run to push changes."
    exit 0
fi

# Commit and push if there are changes
if git diff --quiet; then
    echo "No changes to sync"
else
    git add -A
    git commit -m "Sync from docs/spec/ ($(date -u '+%Y-%m-%d'))"
    git push
    echo ""
    echo "Wiki updated successfully!"
fi
