#!/bin/bash
# Verify a single documentation file against codebase
# Usage: ./scripts/verify/verify-doc.sh <doc-file>

set -e
cd "$(dirname "$0")/../.."

if [ -z "$1" ]; then
  echo "Usage: $0 <documentation-file>"
  echo ""
  echo "Example: $0 docs/spec/error-codes.md"
  exit 1
fi

DOC_FILE="$1"

if [ ! -f "$DOC_FILE" ]; then
  echo "Error: File not found: $DOC_FILE"
  exit 1
fi

echo "========================================"
echo "  Verifying: $DOC_FILE"
echo "========================================"
echo ""

# Extract claims from documentation
echo "=== Extracted Claims ==="
echo ""

# 1. Extract mentioned API endpoints
echo "### API Endpoints Mentioned"
grep -oE '(GET|POST|PUT|PATCH|DELETE) /[a-zA-Z0-9/_\{\}-]+' "$DOC_FILE" 2>/dev/null | sort | uniq || echo "(none found)"
echo ""

# 2. Extract mentioned error codes
echo "### Error Codes Mentioned"
grep -oE '\b[a-z_]+_error\b|\berror_[a-z_]+\b|`[a-z_]+`' "$DOC_FILE" 2>/dev/null | sort | uniq || echo "(none found)"
echo ""

# 3. Extract mentioned environment variables
echo "### Environment Variables Mentioned"
grep -oE '\b[A-Z][A-Z0-9_]{2,}\b' "$DOC_FILE" 2>/dev/null | grep -v "^[A-Z][a-z]" | sort | uniq || echo "(none found)"
echo ""

# 4. Extract code blocks for validation
echo "### Code Blocks (for manual review)"
grep -n '```' "$DOC_FILE" | head -20 || echo "(none found)"
echo ""

# 5. Extract mentioned file paths
echo "### File Paths Mentioned"
grep -oE '[a-z_/]+\.(go|yaml|json|ts|tsx)' "$DOC_FILE" 2>/dev/null | sort | uniq || echo "(none found)"
echo ""

# 6. Check for existence of referenced items
echo "=== Existence Verification ==="
echo ""

# Check API endpoints against code
echo "### Verifying Endpoints..."
for endpoint in $(grep -oE '(GET|POST|PUT|PATCH|DELETE) /[a-zA-Z0-9/_-]+' "$DOC_FILE" 2>/dev/null | sed 's/ /\t/' | cut -f2 | sort | uniq); do
  # Convert path to search pattern (handle {id} style params)
  pattern=$(echo "$endpoint" | sed 's/{[^}]*}/[^/]+/g')
  if grep -rq "\"$pattern\"" adapters/http/ 2>/dev/null; then
    echo "  [OK] $endpoint"
  else
    echo "  [??] $endpoint - not found in adapters/http/"
  fi
done
echo ""

# Check mentioned files exist
echo "### Verifying File References..."
for filepath in $(grep -oE '[a-z_/]+\.(go|yaml|json)' "$DOC_FILE" 2>/dev/null | sort | uniq); do
  if [ -f "$filepath" ]; then
    echo "  [OK] $filepath"
  else
    # Try with common prefixes
    found=false
    for prefix in "" "core/" "adapters/" "pkg/"; do
      if [ -f "${prefix}${filepath}" ]; then
        echo "  [OK] $filepath (at ${prefix}${filepath})"
        found=true
        break
      fi
    done
    if [ "$found" = false ]; then
      echo "  [??] $filepath - not found"
    fi
  fi
done
echo ""

echo "========================================"
echo "  Verification Summary"
echo "========================================"
echo ""
echo "Checked: $DOC_FILE"
echo "Date: $(date)"
echo ""
echo "Legend:"
echo "  [OK] - Verified exists in codebase"
echo "  [??] - Could not verify, needs manual check"
