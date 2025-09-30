#!/bin/bash

# YAML Key Processing Script
# Moves self_service, categories, labels_exclude_any, labels_include_any keys
# from software YAML files to team YAML files

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Global counters
PROCESSED_TEAMS=0
PROCESSED_PACKAGES=0
ERRORS=0

# Check if yq is installed
check_dependencies() {
    if ! command -v yq &> /dev/null; then
        echo -e "${RED}Error: yq is required but not installed. Please install yq first.${NC}"
        echo "Install with: brew install yq"
        exit 1
    fi
    
    # Check yq version (we need v4+)
    YQ_VERSION=$(yq --version | cut -d' ' -f4 | cut -d'v' -f2 | cut -d'.' -f1)
    if [ "$YQ_VERSION" -lt 4 ]; then
        echo -e "${RED}Error: yq version 4 or higher is required${NC}"
        exit 1
    fi
}

# Create backup of a file
backup_file() {
    local file="$1"
    cp "$file" "${file}.bak"
    echo -e "${BLUE}Created backup: ${file}.bak${NC}"
}

# Validate YAML syntax
validate_yaml() {
    local file="$1"
    if ! yq eval '.' "$file" >/dev/null 2>&1; then
        echo -e "${RED}Error: Invalid YAML syntax in $file${NC}"
        return 1
    fi
    return 0
}

# Extract target keys from software file
extract_keys_from_software() {
    local software_file="$1"
    local temp_file=$(mktemp)
    
    # Extract the keys we need
    {
        echo "# Extracted keys from $software_file"
        yq eval 'pick(["self_service", "categories", "labels_include_any", "labels_exclude_any"])' "$software_file" 2>/dev/null || echo "{}"
    } > "$temp_file"
    
    echo "$temp_file"
}

# Remove target keys from software file
remove_keys_from_software() {
    local software_file="$1"
    
    echo -e "${BLUE}  Removing keys from: $software_file${NC}"
    
    # Create a temporary file with keys removed
    local temp_file=$(mktemp)
    yq eval 'del(.self_service, .categories, .labels_include_any, .labels_exclude_any)' "$software_file" > "$temp_file"
    
    # Replace the original file
    mv "$temp_file" "$software_file"
}

# Add keys to team file at specific package index
add_keys_to_team_file() {
    local team_file="$1"
    local package_index="$2"
    local keys_file="$3"
    
    # Check if keys file has any meaningful content
    if ! yq eval 'keys | length > 0' "$keys_file" >/dev/null 2>&1; then
        echo -e "${YELLOW}  No keys to move${NC}"
        return 0
    fi
    
    echo -e "${BLUE}  Adding keys to team file at package index $package_index${NC}"
    
    # Read each key and add it to the team file
    local temp_team_file=$(mktemp)
    cp "$team_file" "$temp_team_file"
    
    # Process each key type
    for key in "self_service" "categories" "labels_include_any" "labels_exclude_any"; do
        if yq eval "has(\"$key\")" "$keys_file" | grep -q "true"; then
            # Use yq to properly extract and merge the value, preserving arrays and complex structures
            if yq eval ".$key != null" "$keys_file" | grep -q "true"; then
                yq eval ".software.packages[$package_index].$key = load(\"$keys_file\").$key" "$temp_team_file" > "${temp_team_file}.tmp"
                mv "${temp_team_file}.tmp" "$temp_team_file"
                echo -e "${GREEN}    Added $key${NC}"
            fi
        fi
    done
    
    # Replace the original team file
    mv "$temp_team_file" "$team_file"
}

# Process a single team file
process_team_file() {
    local team_file="$1"
    echo -e "${GREEN}Processing team file: $team_file${NC}"
    
    # Check if file has software.packages section
    if ! yq eval 'has("software") and .software | has("packages")' "$team_file" | grep -q "true"; then
        echo -e "${YELLOW}  No software.packages section found, skipping${NC}"
        return 0
    fi
    
    # Create backup
    backup_file "$team_file"
    
    # Get the number of packages
    local package_count=$(yq eval '.software.packages | length' "$team_file")
    echo -e "${BLUE}  Found $package_count packages${NC}"
    
    # Process each package
    for ((i=0; i<package_count; i++)); do
        echo -e "${BLUE}  Processing package $((i+1))/$package_count${NC}"
        
        # Get the path from the package
        local package_path=$(yq eval ".software.packages[$i].path" "$team_file")
        
        if [ "$package_path" = "null" ]; then
            echo -e "${YELLOW}    No path found, skipping${NC}"
            continue
        fi
        
        echo -e "${BLUE}    Package path: $package_path${NC}"
        
        # Convert relative path to absolute path
        local team_dir=$(dirname "$team_file")
        local software_file="$team_dir/$package_path"
        
        # Normalize the path
        software_file=$(realpath "$software_file" 2>/dev/null || echo "$software_file")
        
        if [ ! -f "$software_file" ]; then
            echo -e "${RED}    Error: Software file not found: $software_file${NC}"
            ((ERRORS++))
            continue
        fi
        
        # Validate software file
        if ! validate_yaml "$software_file"; then
            echo -e "${RED}    Error: Invalid YAML in software file${NC}"
            ((ERRORS++))
            continue
        fi
        
        echo -e "${BLUE}    Processing: $software_file${NC}"
        
        # Create backup of software file
        backup_file "$software_file"
        
        # Extract keys from software file
        local keys_temp_file=$(extract_keys_from_software "$software_file")
        
        # Add keys to team file
        add_keys_to_team_file "$team_file" "$i" "$keys_temp_file"
        
        # Remove keys from software file
        remove_keys_from_software "$software_file"
        
        # Clean up temp file
        rm -f "$keys_temp_file"
        
        ((PROCESSED_PACKAGES++))
        echo -e "${GREEN}    ✓ Package processed successfully${NC}"
    done
    
    # Validate the modified team file
    if ! validate_yaml "$team_file"; then
        echo -e "${RED}  Error: Team file became invalid after processing${NC}"
        echo -e "${YELLOW}  Restoring from backup...${NC}"
        mv "${team_file}.bak" "$team_file"
        ((ERRORS++))
        return 1
    fi
    
    ((PROCESSED_TEAMS++))
    echo -e "${GREEN}✓ Team file processed successfully${NC}"
    echo
}

# Main function
main() {
    # Accept any script arguments (fixes shellcheck warning)
    local args=("$@")
    
    echo -e "${GREEN}YAML Key Processing Script${NC}"
    echo -e "${BLUE}Moving keys from software files to team files${NC}"
    echo

    # Check dependencies
    check_dependencies
    
    # Check if teams directory exists
    if [ ! -d "it-and-security/teams" ]; then
        echo -e "${RED}Error: it-and-security/teams directory not found${NC}"
        echo "Please run this script from the fleet repository root"
        exit 1
    fi
    
    # Process all YAML files in teams directory
    echo -e "${BLUE}Finding team files...${NC}"
    
    # Use nullglob to handle case where no files match the pattern
    shopt -s nullglob
    local team_files=(it-and-security/teams/*.yml)
    shopt -u nullglob
    
    if [ ${#team_files[@]} -eq 0 ]; then
        echo -e "${RED}Error: No YAML files found in teams directory${NC}"
        exit 1
    fi
    
    echo -e "${BLUE}Found ${#team_files[@]} team files${NC}"
    echo
    
    # Process each team file
    for team_file in "${team_files[@]}"; do
        if [ -f "$team_file" ]; then
            process_team_file "$team_file"
        fi
    done
    
    # Summary
    echo -e "${GREEN}=== PROCESSING COMPLETE ===${NC}"
    echo -e "${GREEN}Teams processed: $PROCESSED_TEAMS${NC}"
    echo -e "${GREEN}Packages processed: $PROCESSED_PACKAGES${NC}"
    if [ $ERRORS -gt 0 ]; then
        echo -e "${RED}Errors encountered: $ERRORS${NC}"
        echo -e "${YELLOW}Check the output above for details${NC}"
    else
        echo -e "${GREEN}✓ All files processed successfully!${NC}"
    fi
    
    echo
    echo -e "${BLUE}Backup files created with .bak extension${NC}"
    echo -e "${BLUE}To restore from backups if needed: find . -name '*.bak' -exec bash -c 'mv \"$1\" \"${1%.bak}\"' _ {} \\;${NC}"
}

# Run main function with all script arguments
main "$@"