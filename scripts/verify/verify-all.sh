#!/bin/bash
# Documentation Verification Suite
# Run from project root: ./scripts/verify/verify-all.sh

set -e
cd "$(dirname "$0")/../.."

echo "========================================"
echo "  APIGate Documentation Verification"
echo "========================================"
echo ""

REPORT_DIR="docs/verification-reports"
mkdir -p "$REPORT_DIR"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
REPORT_FILE="$REPORT_DIR/report_$TIMESTAMP.md"

echo "# Verification Report - $(date)" > "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# ============================================
# 1. API Endpoints Verification
# ============================================
echo "## 1. API Endpoints" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

echo "[1/7] Checking API endpoints..."

echo "### Routes in Code" >> "$REPORT_FILE"
echo '```' >> "$REPORT_FILE"
grep -rn "router\.\(GET\|POST\|PUT\|PATCH\|DELETE\)" adapters/http/admin/*.go 2>/dev/null | \
  grep -v "_test.go" | \
  sed 's/.*:.*"\([^"]*\)".*/\1/' | sort | uniq >> "$REPORT_FILE" || echo "No routes found" >> "$REPORT_FILE"
echo '```' >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# ============================================
# 2. Error Codes Verification
# ============================================
echo "## 2. Error Codes" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

echo "[2/7] Checking error codes..."

echo "### Error Codes in Code (pkg/jsonapi/errors.go)" >> "$REPORT_FILE"
echo '```' >> "$REPORT_FILE"
grep -E "NewError\([0-9]+," pkg/jsonapi/errors.go 2>/dev/null | \
  sed 's/.*NewError(\([0-9]*\), "\([^"]*\)", "\([^"]*\)".*/| \1 | \2 | \3 |/' >> "$REPORT_FILE" || echo "No errors found" >> "$REPORT_FILE"
echo '```' >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# Check if error-codes.md exists and compare
if [ -f "docs/spec/error-codes.md" ]; then
  echo "### Error Codes in Docs" >> "$REPORT_FILE"
  echo '```' >> "$REPORT_FILE"
  grep -E "^\| [0-9]+" docs/spec/error-codes.md >> "$REPORT_FILE" || echo "No error table found" >> "$REPORT_FILE"
  echo '```' >> "$REPORT_FILE"
fi
echo "" >> "$REPORT_FILE"

# ============================================
# 3. Module YAML Verification
# ============================================
echo "## 3. Module YAMLs" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

echo "[3/7] Checking module definitions..."

echo "### Core Modules" >> "$REPORT_FILE"
echo '```' >> "$REPORT_FILE"
ls core/modules/*.yaml 2>/dev/null | xargs -I{} basename {} .yaml >> "$REPORT_FILE" || echo "No modules found" >> "$REPORT_FILE"
echo '```' >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

echo "### Capabilities" >> "$REPORT_FILE"
echo '```' >> "$REPORT_FILE"
ls core/modules/capabilities/*.yaml 2>/dev/null | xargs -I{} basename {} .yaml >> "$REPORT_FILE" || echo "No capabilities found" >> "$REPORT_FILE"
echo '```' >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

echo "### Providers" >> "$REPORT_FILE"
echo '```' >> "$REPORT_FILE"
ls core/modules/providers/*.yaml 2>/dev/null | xargs -I{} basename {} .yaml >> "$REPORT_FILE" || echo "No providers found" >> "$REPORT_FILE"
echo '```' >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# ============================================
# 4. Environment Variables
# ============================================
echo "## 4. Environment Variables" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

echo "[4/7] Checking environment variables..."

echo "### Env Vars in Code" >> "$REPORT_FILE"
echo '```' >> "$REPORT_FILE"
grep -rh "os\.Getenv\|viper\." --include="*.go" . 2>/dev/null | \
  grep -oE '"[A-Z][A-Z0-9_]*"' | tr -d '"' | sort | uniq >> "$REPORT_FILE" || echo "None found" >> "$REPORT_FILE"
echo '```' >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# ============================================
# 5. Handler Functions
# ============================================
echo "## 5. Handler Functions" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

echo "[5/7] Checking handler functions..."

echo "### Admin Handlers" >> "$REPORT_FILE"
echo '```' >> "$REPORT_FILE"
grep -rn "^func.*Handler\|^func Handle" adapters/http/admin/*.go 2>/dev/null | \
  grep -v "_test.go" | \
  sed 's/.*:\(func [^(]*\).*/\1/' >> "$REPORT_FILE" || echo "None found" >> "$REPORT_FILE"
echo '```' >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# ============================================
# 6. Documentation Files Inventory
# ============================================
echo "## 6. Documentation Inventory" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

echo "[6/7] Building documentation inventory..."

echo "### docs/spec/" >> "$REPORT_FILE"
echo '```' >> "$REPORT_FILE"
find docs/spec -name "*.md" -type f 2>/dev/null | wc -l | xargs echo "Total files:" >> "$REPORT_FILE"
find docs/spec -name "*.md" -type f 2>/dev/null >> "$REPORT_FILE" || echo "None found" >> "$REPORT_FILE"
echo '```' >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

echo "### docs/user_journeys/" >> "$REPORT_FILE"
echo '```' >> "$REPORT_FILE"
find docs/user_journeys -name "*.md" -type f 2>/dev/null >> "$REPORT_FILE" || echo "None found" >> "$REPORT_FILE"
echo '```' >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# ============================================
# 7. Cross-Reference Check
# ============================================
echo "## 7. Cross-Reference Checks" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

echo "[7/7] Running cross-reference checks..."

echo "### Resource Types - YAML vs Docs" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# List YAML resource types
echo "**YAMLs defined:**" >> "$REPORT_FILE"
echo '```' >> "$REPORT_FILE"
ls core/modules/*.yaml 2>/dev/null | xargs -I{} basename {} .yaml | sort >> "$REPORT_FILE"
echo '```' >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# Check if resource-types.md documents them
if [ -f "docs/spec/resource-types.md" ]; then
  echo "**Documented in resource-types.md:**" >> "$REPORT_FILE"
  echo '```' >> "$REPORT_FILE"
  grep -E "^## |^### " docs/spec/resource-types.md | sed 's/^#* //' >> "$REPORT_FILE"
  echo '```' >> "$REPORT_FILE"
fi

echo "" >> "$REPORT_FILE"

# ============================================
# Summary
# ============================================
echo "## Summary" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"
echo "| Category | Count |" >> "$REPORT_FILE"
echo "|----------|-------|" >> "$REPORT_FILE"

YAML_COUNT=$(ls core/modules/*.yaml core/modules/*/*.yaml 2>/dev/null | wc -l | tr -d ' ')
DOC_COUNT=$(find docs -name "*.md" -type f 2>/dev/null | wc -l | tr -d ' ')
HANDLER_COUNT=$(grep -rn "^func.*Handler" adapters/http/admin/*.go 2>/dev/null | grep -v "_test.go" | wc -l | tr -d ' ')

echo "| Module YAMLs | $YAML_COUNT |" >> "$REPORT_FILE"
echo "| Documentation Files | $DOC_COUNT |" >> "$REPORT_FILE"
echo "| Handler Functions | $HANDLER_COUNT |" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

echo "Generated: $(date)" >> "$REPORT_FILE"

echo ""
echo "========================================"
echo "  Verification Complete"
echo "========================================"
echo ""
echo "Report saved to: $REPORT_FILE"
echo ""
echo "To view: cat $REPORT_FILE"
