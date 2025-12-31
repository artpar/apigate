#!/bin/bash

#
# User Journey Screenshot & GIF Capture Script
#
# This script captures screenshots and generates GIFs for all user journeys
# defined in docs/USER_JOURNEYS.md
#
# Usage:
#   ./scripts/capture-journeys.sh [options] [journey]
#
# Options:
#   --fresh       Use a fresh database (requires restart)
#   --gifs-only   Only generate GIFs from existing frames
#   --no-gifs     Skip GIF generation
#   --help        Show this help
#
# Examples:
#   ./scripts/capture-journeys.sh all          # Capture all journeys
#   ./scripts/capture-journeys.sh j1           # Capture J1: Setup only
#   ./scripts/capture-journeys.sh j5 j6        # Capture J5 and J6
#   ./scripts/capture-journeys.sh --gifs-only  # Generate GIFs only
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Directories
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
WEBUI_DIR="$PROJECT_ROOT/webui"
DOCS_DIR="$PROJECT_ROOT/docs"
SCREENSHOTS_DIR="$DOCS_DIR/screenshots"
GIFS_DIR="$DOCS_DIR/gifs"
GIF_FRAMES_DIR="$DOCS_DIR/.gif-frames"

# Default options
FRESH_DB=false
GIFS_ONLY=false
NO_GIFS=false
JOURNEYS=()

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --fresh)
            FRESH_DB=true
            shift
            ;;
        --gifs-only)
            GIFS_ONLY=true
            shift
            ;;
        --no-gifs)
            NO_GIFS=true
            shift
            ;;
        --help|-h)
            head -30 "$0" | tail -27
            exit 0
            ;;
        all)
            JOURNEYS=("J1" "J2" "J3" "J4" "J5" "J6" "J7" "J8" "J9")
            shift
            ;;
        j[1-9]|J[1-9])
            JOURNEYS+=("${1^^}")
            shift
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Default to all journeys if none specified
if [ ${#JOURNEYS[@]} -eq 0 ] && [ "$GIFS_ONLY" = false ]; then
    JOURNEYS=("J1" "J2" "J3" "J4" "J5" "J6" "J7" "J8" "J9")
fi

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_dependencies() {
    log_info "Checking dependencies..."

    # Check if playwright is installed
    if ! command -v npx &> /dev/null; then
        log_error "npx not found. Please install Node.js."
        exit 1
    fi

    # Check if ffmpeg is available (for GIF generation)
    if ! command -v ffmpeg &> /dev/null; then
        log_warn "ffmpeg not found. GIF generation will be skipped."
        log_warn "Install with: brew install ffmpeg (macOS) or apt install ffmpeg (Linux)"
        NO_GIFS=true
    fi
}

check_server() {
    log_info "Checking if APIGate server is running..."

    if ! curl -s http://localhost:8080/health > /dev/null 2>&1; then
        log_error "APIGate server is not running at localhost:8080"
        log_error "Please start the server: ./apigate serve"
        exit 1
    fi

    log_success "Server is running"
}

setup_directories() {
    log_info "Setting up directories..."

    mkdir -p "$SCREENSHOTS_DIR"
    mkdir -p "$GIFS_DIR"
    mkdir -p "$GIF_FRAMES_DIR"

    # Create journey subdirectories
    for journey in "${JOURNEYS[@]}"; do
        case $journey in
            J1) mkdir -p "$SCREENSHOTS_DIR/j1-setup" ;;
            J2) mkdir -p "$SCREENSHOTS_DIR/j2-plans" ;;
            J3) mkdir -p "$SCREENSHOTS_DIR/j3-monitor" ;;
            J4) mkdir -p "$SCREENSHOTS_DIR/j4-config" ;;
            J5) mkdir -p "$SCREENSHOTS_DIR/j5-onboarding" ;;
            J6) mkdir -p "$SCREENSHOTS_DIR/j6-api-access" ;;
            J7) mkdir -p "$SCREENSHOTS_DIR/j7-usage" ;;
            J8) mkdir -p "$SCREENSHOTS_DIR/j8-upgrade" ;;
            J9) mkdir -p "$SCREENSHOTS_DIR/j9-docs" ;;
        esac
    done

    mkdir -p "$SCREENSHOTS_DIR/errors"
}

capture_journey() {
    local journey=$1
    log_info "Capturing $journey..."

    cd "$WEBUI_DIR"

    # Build grep pattern for the journey
    local pattern=""
    case $journey in
        J1) pattern="J1:" ;;
        J2) pattern="J2:" ;;
        J3) pattern="J3:" ;;
        J4) pattern="J4:" ;;
        J5) pattern="J5:" ;;
        J6) pattern="J6:" ;;
        J7) pattern="J7:" ;;
        J8) pattern="J8:" ;;
        J9) pattern="J9:" ;;
    esac

    # Run playwright for this journey
    npx playwright test capture-journeys.spec.ts \
        --project=chromium \
        --grep "$pattern" \
        --reporter=list \
        2>&1 | while read line; do
            if [[ $line == *"✓ Captured"* ]]; then
                echo -e "  ${GREEN}$line${NC}"
            elif [[ $line == *"✘"* ]] || [[ $line == *"Error"* ]]; then
                echo -e "  ${RED}$line${NC}"
            fi
        done

    log_success "Completed $journey"
}

generate_gif() {
    local journey=$1
    local journey_dir=""
    local output_name=""

    case $journey in
        J1) journey_dir="j1-setup"; output_name="j1-setup-wizard" ;;
        J2) journey_dir="j2-plans"; output_name="j2-create-plan" ;;
        J5) journey_dir="j5-onboarding"; output_name="j5-signup" ;;
        J6) journey_dir="j6-api-access"; output_name="j6-create-key" ;;
        J8) journey_dir="j8-upgrade"; output_name="j8-upgrade" ;;
        J9) journey_dir="j9-docs"; output_name="j9-docs-tour" ;;
        *) return ;;
    esac

    local frames_path="$SCREENSHOTS_DIR/$journey_dir"
    local output_path="$GIFS_DIR/${output_name}.gif"

    # Check if frames exist
    local frame_count=$(ls -1 "$frames_path"/*.png 2>/dev/null | wc -l)
    if [ "$frame_count" -lt 2 ]; then
        log_warn "Not enough frames for $journey GIF (found $frame_count)"
        return
    fi

    log_info "Generating GIF: $output_name.gif ($frame_count frames)"

    # Generate GIF using ffmpeg
    # -framerate 0.5 = 2 seconds per frame
    # -vf scale=1280:-1 = scale to 1280px width
    ffmpeg -y \
        -framerate 0.5 \
        -pattern_type glob \
        -i "$frames_path/*.png" \
        -vf "scale=1280:-1:flags=lanczos,split[s0][s1];[s0]palettegen=max_colors=128[p];[s1][p]paletteuse=dither=bayer" \
        "$output_path" \
        2>/dev/null

    if [ -f "$output_path" ]; then
        local size=$(du -h "$output_path" | cut -f1)
        log_success "Created $output_name.gif ($size)"
    else
        log_error "Failed to create $output_name.gif"
    fi
}

generate_all_gifs() {
    if [ "$NO_GIFS" = true ]; then
        log_info "Skipping GIF generation (--no-gifs)"
        return
    fi

    log_info "Generating GIFs..."

    for journey in J1 J2 J5 J6 J8 J9; do
        generate_gif "$journey"
    done
}

main() {
    echo ""
    echo "=========================================="
    echo "  APIGate User Journey Capture"
    echo "=========================================="
    echo ""

    check_dependencies

    if [ "$GIFS_ONLY" = true ]; then
        generate_all_gifs
        exit 0
    fi

    check_server
    setup_directories

    log_info "Capturing journeys: ${JOURNEYS[*]}"
    echo ""

    for journey in "${JOURNEYS[@]}"; do
        capture_journey "$journey"
        echo ""
    done

    # Generate GIFs after all captures
    generate_all_gifs

    echo ""
    echo "=========================================="
    echo "  Capture Complete!"
    echo "=========================================="
    echo ""
    echo "Screenshots: $SCREENSHOTS_DIR"
    echo "GIFs: $GIFS_DIR"
    echo ""

    # Summary
    local total_screenshots=$(find "$SCREENSHOTS_DIR" -name "*.png" | wc -l | tr -d ' ')
    local total_gifs=$(find "$GIFS_DIR" -name "*.gif" 2>/dev/null | wc -l | tr -d ' ')

    echo "Total screenshots: $total_screenshots"
    echo "Total GIFs: $total_gifs"
}

main "$@"
