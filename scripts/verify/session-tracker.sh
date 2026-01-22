#!/bin/bash
# Session Tracker for Documentation Verification
# Maintains state across multiple verification sessions
#
# Usage:
#   ./session-tracker.sh start "Section 1"     - Start new session
#   ./session-tracker.sh complete "file.md"    - Mark file as verified
#   ./session-tracker.sh issue "file.md" "msg" - Log an issue
#   ./session-tracker.sh status                - Show progress
#   ./session-tracker.sh next                  - Show next unverified files

set -e
cd "$(dirname "$0")/../.."

STATE_FILE="docs/verification-reports/verification-state.json"
mkdir -p "$(dirname "$STATE_FILE")"

# Initialize state file if doesn't exist
if [ ! -f "$STATE_FILE" ]; then
  cat > "$STATE_FILE" << 'EOF'
{
  "last_updated": "",
  "sessions": [],
  "verified_files": [],
  "issues": [],
  "current_session": null
}
EOF
fi

case "$1" in
  start)
    SESSION_NAME="${2:-$(date +%Y-%m-%d)}"
    echo "Starting verification session: $SESSION_NAME"

    # Update state with jq if available, otherwise manual
    if command -v jq &> /dev/null; then
      jq --arg name "$SESSION_NAME" --arg date "$(date -Iseconds)" \
        '.current_session = $name | .sessions += [{name: $name, started: $date}] | .last_updated = $date' \
        "$STATE_FILE" > "${STATE_FILE}.tmp" && mv "${STATE_FILE}.tmp" "$STATE_FILE"
    else
      echo "Session started: $SESSION_NAME at $(date)" >> "$STATE_FILE.log"
    fi
    echo "Session started. Use 'complete' to mark files as verified."
    ;;

  complete)
    FILE="$2"
    if [ -z "$FILE" ]; then
      echo "Usage: $0 complete <filename>"
      exit 1
    fi

    if command -v jq &> /dev/null; then
      jq --arg file "$FILE" --arg date "$(date -Iseconds)" \
        '.verified_files += [{file: $file, verified_at: $date}] | .last_updated = $date' \
        "$STATE_FILE" > "${STATE_FILE}.tmp" && mv "${STATE_FILE}.tmp" "$STATE_FILE"
    else
      echo "VERIFIED: $FILE at $(date)" >> "$STATE_FILE.log"
    fi
    echo "Marked as verified: $FILE"
    ;;

  issue)
    FILE="$2"
    MESSAGE="$3"
    SEVERITY="${4:-medium}"

    if [ -z "$FILE" ] || [ -z "$MESSAGE" ]; then
      echo "Usage: $0 issue <filename> <message> [severity]"
      exit 1
    fi

    if command -v jq &> /dev/null; then
      jq --arg file "$FILE" --arg msg "$MESSAGE" --arg sev "$SEVERITY" --arg date "$(date -Iseconds)" \
        '.issues += [{file: $file, message: $msg, severity: $sev, found_at: $date}] | .last_updated = $date' \
        "$STATE_FILE" > "${STATE_FILE}.tmp" && mv "${STATE_FILE}.tmp" "$STATE_FILE"
    else
      echo "ISSUE [$SEVERITY]: $FILE - $MESSAGE at $(date)" >> "$STATE_FILE.log"
    fi
    echo "Issue logged for: $FILE"
    ;;

  status)
    echo "========================================"
    echo "  Verification Progress"
    echo "========================================"
    echo ""

    # Count all docs
    TOTAL_DOCS=$(find docs -name "*.md" -type f 2>/dev/null | wc -l | tr -d ' ')
    TOTAL_YAMLS=$(find core/modules -name "*.yaml" -type f 2>/dev/null | wc -l | tr -d ' ')
    TOTAL=$((TOTAL_DOCS + TOTAL_YAMLS))

    if command -v jq &> /dev/null && [ -f "$STATE_FILE" ]; then
      VERIFIED=$(jq '.verified_files | length' "$STATE_FILE")
      ISSUES=$(jq '.issues | length' "$STATE_FILE")
      SESSIONS=$(jq '.sessions | length' "$STATE_FILE")
      LAST_UPDATE=$(jq -r '.last_updated' "$STATE_FILE")
    else
      VERIFIED=$(grep -c "^VERIFIED:" "$STATE_FILE.log" 2>/dev/null || echo 0)
      ISSUES=$(grep -c "^ISSUE" "$STATE_FILE.log" 2>/dev/null || echo 0)
      SESSIONS="unknown"
      LAST_UPDATE="unknown"
    fi

    echo "Total items to verify: $TOTAL"
    echo "  - Documentation files: $TOTAL_DOCS"
    echo "  - Module YAMLs: $TOTAL_YAMLS"
    echo ""
    echo "Verified: $VERIFIED"
    echo "Issues found: $ISSUES"
    echo "Sessions: $SESSIONS"
    echo "Last update: $LAST_UPDATE"
    echo ""

    if command -v jq &> /dev/null && [ -f "$STATE_FILE" ]; then
      PCTG=$((VERIFIED * 100 / TOTAL))
      echo "Progress: $PCTG%"
      echo ""

      if [ "$ISSUES" -gt 0 ]; then
        echo "Recent issues:"
        jq -r '.issues[-5:] | .[] | "  [\(.severity)] \(.file): \(.message)"' "$STATE_FILE"
      fi
    fi
    ;;

  next)
    echo "========================================"
    echo "  Next Files to Verify"
    echo "========================================"
    echo ""

    # Get verified files list
    if command -v jq &> /dev/null && [ -f "$STATE_FILE" ]; then
      VERIFIED_LIST=$(jq -r '.verified_files[].file' "$STATE_FILE" 2>/dev/null | sort)
    else
      VERIFIED_LIST=$(grep "^VERIFIED:" "$STATE_FILE.log" 2>/dev/null | cut -d: -f2 | tr -d ' ' | sort)
    fi

    echo "Priority: API Specification (docs/spec/)"
    for f in docs/spec/*.md; do
      if [ -f "$f" ] && ! echo "$VERIFIED_LIST" | grep -q "^$f$"; then
        echo "  [ ] $f"
      fi
    done
    echo ""

    echo "Priority: Wiki (docs/spec/wiki/) - first 10 unverified"
    count=0
    for f in docs/spec/wiki/*.md; do
      if [ -f "$f" ] && ! echo "$VERIFIED_LIST" | grep -q "^$f$"; then
        echo "  [ ] $f"
        count=$((count + 1))
        if [ $count -ge 10 ]; then
          echo "  ... and more"
          break
        fi
      fi
    done
    echo ""

    echo "Priority: User Journeys"
    for f in docs/user_journeys/**/*.md; do
      if [ -f "$f" ] && ! echo "$VERIFIED_LIST" | grep -q "^$f$"; then
        echo "  [ ] $f"
      fi
    done 2>/dev/null
    ;;

  report)
    echo "Generating summary report..."

    REPORT="docs/verification-reports/summary_$(date +%Y%m%d).md"

    echo "# Verification Summary - $(date +%Y-%m-%d)" > "$REPORT"
    echo "" >> "$REPORT"

    if command -v jq &> /dev/null && [ -f "$STATE_FILE" ]; then
      echo "## Progress" >> "$REPORT"
      VERIFIED=$(jq '.verified_files | length' "$STATE_FILE")
      TOTAL=$(find docs -name "*.md" -type f | wc -l | tr -d ' ')
      echo "- Verified: $VERIFIED / $TOTAL" >> "$REPORT"
      echo "" >> "$REPORT"

      echo "## Issues Found" >> "$REPORT"
      echo "" >> "$REPORT"
      jq -r '.issues[] | "- **\(.file)** [\(.severity)]: \(.message)"' "$STATE_FILE" >> "$REPORT"
      echo "" >> "$REPORT"

      echo "## Verified Files" >> "$REPORT"
      echo "" >> "$REPORT"
      jq -r '.verified_files[] | "- [x] \(.file) (\(.verified_at))"' "$STATE_FILE" >> "$REPORT"
    fi

    echo "Report saved to: $REPORT"
    ;;

  *)
    echo "Documentation Verification Session Tracker"
    echo ""
    echo "Usage: $0 <command> [args]"
    echo ""
    echo "Commands:"
    echo "  start [name]              Start a new verification session"
    echo "  complete <file>           Mark a file as verified"
    echo "  issue <file> <msg> [sev]  Log an issue (sev: low/medium/high/critical)"
    echo "  status                    Show overall progress"
    echo "  next                      Show next unverified files"
    echo "  report                    Generate summary report"
    ;;
esac
