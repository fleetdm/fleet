#!/bin/bash

set -e

# Function to display usage
usage() {
    echo "Usage: $0 <file_path>"
    echo "Example: $0 ee/maintained-apps/outputs/adobe-acrobat-reader/darwin.json"
    echo ""
    echo "This script will:"
    echo "1. Find changes in install_script_ref and uninstall_script_ref between current and previous commit"
    echo "2. Extract the actual scripts from the refs section"
    echo "3. Show a diff of the scripts"
    exit 1
}

# Check if file path is provided
if [ $# -ne 1 ]; then
    usage
fi

FILE_PATH="$1"

# Check if file exists
if [ ! -f "$FILE_PATH" ]; then
    echo "Error: File '$FILE_PATH' does not exist"
    exit 1
fi

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo "Error: Not in a git repository"
    exit 1
fi

# Function to extract script content from refs
extract_script() {
    local json_content="$1"
    local ref_id="$2"
    
    if [ -z "$ref_id" ] || [ "$ref_id" = "null" ]; then
        echo ""
        return
    fi
    
    echo "$json_content" | jq -r ".refs[\"$ref_id\"] // empty" 2>/dev/null || echo ""
}

# Get current file content
echo "Analyzing changes in: $FILE_PATH"
echo "=================================="

# Get current and previous file content
CURRENT_CONTENT=$(cat "$FILE_PATH")
PREVIOUS_CONTENT=$(git show HEAD~1:"$FILE_PATH" 2>/dev/null || echo "")

if [ -z "$PREVIOUS_CONTENT" ]; then
    echo "Warning: Could not retrieve previous version of file (file may not exist in previous commit)"
    echo "Current file content will be shown without comparison"

    # Show current refs
    CURRENT_INSTALL_REF=$(echo "$CURRENT_CONTENT" | jq -r '.versions[0].install_script_ref // empty')
    CURRENT_UNINSTALL_REF=$(echo "$CURRENT_CONTENT" | jq -r '.versions[0].uninstall_script_ref // empty')

    echo ""
    echo "Current install_script_ref: $CURRENT_INSTALL_REF"
    echo "Current uninstall_script_ref: $CURRENT_UNINSTALL_REF"

    if [ -n "$CURRENT_INSTALL_REF" ] && [ "$CURRENT_INSTALL_REF" != "null" ]; then
        echo ""
        echo "=== Current Install Script (ref: $CURRENT_INSTALL_REF) ==="
        extract_script "$CURRENT_CONTENT" "$CURRENT_INSTALL_REF"
    fi

    if [ -n "$CURRENT_UNINSTALL_REF" ] && [ "$CURRENT_UNINSTALL_REF" != "null" ]; then
        echo ""
        echo "=== Current Uninstall Script (ref: $CURRENT_UNINSTALL_REF) ==="
        extract_script "$CURRENT_CONTENT" "$CURRENT_UNINSTALL_REF"
    fi

    exit 0
fi

# Extract script references from both versions
CURRENT_INSTALL_REF=$(echo "$CURRENT_CONTENT" | jq -r '.versions[0].install_script_ref // empty')
CURRENT_UNINSTALL_REF=$(echo "$CURRENT_CONTENT" | jq -r '.versions[0].uninstall_script_ref // empty')

PREVIOUS_INSTALL_REF=$(echo "$PREVIOUS_CONTENT" | jq -r '.versions[0].install_script_ref // empty')
PREVIOUS_UNINSTALL_REF=$(echo "$PREVIOUS_CONTENT" | jq -r '.versions[0].uninstall_script_ref // empty')

echo ""
echo "Script reference changes:"
echo "Install script:   $PREVIOUS_INSTALL_REF -> $CURRENT_INSTALL_REF"
echo "Uninstall script: $PREVIOUS_UNINSTALL_REF -> $CURRENT_UNINSTALL_REF"

# Function to show script diff
show_script_diff() {
    local script_type="$1"
    local prev_ref="$2"
    local curr_ref="$3"
    local prev_content="$4"
    local curr_content="$5"
    
    if [ "$prev_ref" = "$curr_ref" ]; then
        echo ""
        echo "=== $script_type Script (no changes) ==="
        echo "Reference: $curr_ref"
        return
    fi
    
    echo ""
    echo "=== $script_type Script Diff ==="
    echo "Previous ref: $prev_ref"
    echo "Current ref:  $curr_ref"
    echo ""
    
    # Extract scripts
    PREV_SCRIPT=$(extract_script "$prev_content" "$prev_ref")
    CURR_SCRIPT=$(extract_script "$curr_content" "$curr_ref")
    
    if [ -z "$PREV_SCRIPT" ] && [ -z "$CURR_SCRIPT" ]; then
        echo "No scripts found for either version"
        return
    elif [ -z "$PREV_SCRIPT" ]; then
        echo "Previous script not found, showing current script:"
        echo "$CURR_SCRIPT"
        return
    elif [ -z "$CURR_SCRIPT" ]; then
        echo "Current script not found, showing previous script:"
        echo "$PREV_SCRIPT"
        return
    fi
    
    # Create temporary files for diff
    TEMP_PREV=$(mktemp)
    TEMP_CURR=$(mktemp)
    
    echo "$PREV_SCRIPT" > "$TEMP_PREV"
    echo "$CURR_SCRIPT" > "$TEMP_CURR"
    
    # Show unified diff
    if diff -u "$TEMP_PREV" "$TEMP_CURR" > /dev/null 2>&1; then
        echo "No differences found in script content"
    else
        diff -u "$TEMP_PREV" "$TEMP_CURR" || true
    fi
    
    # Clean up temp files
    rm -f "$TEMP_PREV" "$TEMP_CURR"
}

# Show install script diff
if [ -n "$CURRENT_INSTALL_REF" ] || [ -n "$PREVIOUS_INSTALL_REF" ]; then
    show_script_diff "Install" "$PREVIOUS_INSTALL_REF" "$CURRENT_INSTALL_REF" "$PREVIOUS_CONTENT" "$CURRENT_CONTENT"
fi

# Show uninstall script diff
if [ -n "$CURRENT_UNINSTALL_REF" ] || [ -n "$PREVIOUS_UNINSTALL_REF" ]; then
    show_script_diff "Uninstall" "$PREVIOUS_UNINSTALL_REF" "$CURRENT_UNINSTALL_REF" "$PREVIOUS_CONTENT" "$CURRENT_CONTENT"
fi

echo ""
echo "Done."
