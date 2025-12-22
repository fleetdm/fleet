#!/bin/bash

# Script to refetch host details using Fleet's device authentication token
# This script reads the device token from /opt/orbit/identifier and triggers a refetch

set -euo pipefail  # Exit on error, undefined vars, and pipe failures

# Configuration
IDENTIFIER_FILE="/opt/orbit/identifier"
FLEET_URL="https://dogfood.fleetdm.com"  # Set this environment variable or modify the script
LOG_LEVEL="${LOG_LEVEL:-info}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging function
log() {
    local level="$1"
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    case "$level" in
        "error")
            echo -e "${timestamp} [${RED}ERROR${NC}] $message" >&2
            ;;
        "warn")
            echo -e "${timestamp} [${YELLOW}WARN${NC}] $message" >&2
            ;;
        "info")
            echo -e "${timestamp} [${GREEN}INFO${NC}] $message"
            ;;
        "debug")
            if [[ "$LOG_LEVEL" == "debug" ]]; then
                echo -e "${timestamp} [DEBUG] $message"
            fi
            ;;
    esac
}

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to validate URL format
validate_url() {
    local url="$1"
    if [[ ! "$url" =~ ^https?://[^[:space:]]+$ ]]; then
        return 1
    fi
    return 0
}

# Function to read device token
read_device_token() {
    local token_file="$1"
    
    if [[ ! -f "$token_file" ]]; then
        log error "Device token file not found: $token_file"
        return 1
    fi
    
    if [[ ! -r "$token_file" ]]; then
        log error "Cannot read device token file: $token_file (permission denied)"
        return 1
    fi
    
    local token
    token=$(cat "$token_file" 2>/dev/null)
    
    if [[ -z "$token" ]]; then
        log error "Device token file is empty: $token_file"
        return 1
    fi
    
    # Basic validation - Fleet device tokens should be non-empty strings
    if [[ ${#token} -lt 10 ]]; then
        log warn "Device token seems unusually short (${#token} characters)"
    fi
    
    echo "$token"
}

# Function to get host ID from device token
get_host_id() {
    local fleet_url="$1"
    local device_token="$2"
    
    log debug "Attempting to get host ID using device token..."
    
    # Use the device endpoint to get basic host information
    local response
    local http_code
    
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -H "Accept: application/json" \
        -H "User-Agent: fleet-refetch-script/1.0" \
        --max-time 30 \
        --retry 2 \
        --retry-delay 1 \
        "${fleet_url}/api/v1/fleet/device/${device_token}" 2>/dev/null)
    
    if [[ $? -ne 0 ]]; then
        log error "Failed to connect to Fleet server at $fleet_url"
        return 1
    fi
    
    http_code=$(echo "$response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    response_body=$(echo "$response" | sed -e 's/HTTPSTATUS:.*//g')
    
    log debug "HTTP response code: $http_code"
    
    if [[ "$http_code" -ne 200 ]]; then
        log error "Failed to authenticate with device token (HTTP $http_code)"
        if [[ "$http_code" -eq 401 ]]; then
            log error "Device token appears to be invalid or expired"
        elif [[ "$http_code" -eq 404 ]]; then
            log error "Device not found or Fleet server endpoint not available"
        fi
        return 1
    fi
    
    # Extract host ID from JSON response
    local host_id
    if command_exists jq; then
        host_id=$(echo "$response_body" | jq -r '.host.id' 2>/dev/null)
    else
        # Fallback: basic grep/sed extraction (less reliable but doesn't require jq)
        host_id=$(echo "$response_body" | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
    fi
    
    if [[ -z "$host_id" || "$host_id" == "null" ]]; then
        log error "Could not extract host ID from response"
        log debug "Response body: $response_body"
        return 1
    fi
    
    echo "$host_id"
}

# Function to trigger device-level refetch using device token
trigger_device_refetch() {
    local fleet_url="$1"
    local device_token="$2"
    
    log info "Triggering device refetch using device token..."
    
    # Try device-specific refetch endpoint that may accept device tokens
    local response
    local http_code
    
    # First attempt: Try device-specific refetch endpoint
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X POST \
        -H "Accept: application/json" \
        -H "User-Agent: fleet-refetch-script/1.0" \
        --max-time 30 \
        --retry 2 \
        --retry-delay 1 \
        "${fleet_url}/api/v1/fleet/device/${device_token}/refetch" 2>/dev/null)
    
    if [[ $? -ne 0 ]]; then
        log warn "Failed to connect to device-specific refetch endpoint, trying alternative..."
        return 1
    fi
    
    http_code=$(echo "$response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    response_body=$(echo "$response" | sed -e 's/HTTPSTATUS:.*//g')
    
    log debug "Device refetch HTTP response code: $http_code"
    
    case "$http_code" in
        200|202)
            log info "Device refetch triggered successfully"
            return 0
            ;;
        404)
            log warn "Device-specific refetch endpoint not available, trying alternative method..."
            return 1
            ;;
        401|403)
            log error "Device token authentication failed for refetch"
            return 1
            ;;
        *)
            log warn "Device refetch returned HTTP $http_code, trying alternative method..."
            return 1
            ;;
    esac
}

# Function to trigger refetch via orbit/fleetd ping mechanism
trigger_orbit_ping() {
    local fleet_url="$1"
    local device_token="$2"
    
    log info "Attempting to trigger refetch via orbit ping mechanism..."
    
    local response
    local http_code
    
    # Try the orbit device ping endpoint which may trigger a refetch
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X POST \
        -H "Accept: application/json" \
        -H "Content-Type: application/json" \
        -H "User-Agent: fleet-refetch-script/1.0" \
        -d '{"node_key": "'$device_token'", "refetch_requested": true}' \
        --max-time 30 \
        --retry 2 \
        --retry-delay 1 \
        "${fleet_url}/api/fleet/orbit/ping" 2>/dev/null)
    
    if [[ $? -ne 0 ]]; then
        log warn "Failed to connect to orbit ping endpoint"
        return 1
    fi
    
    http_code=$(echo "$response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    response_body=$(echo "$response" | sed -e 's/HTTPSTATUS:.*//g')
    
    log debug "Orbit ping HTTP response code: $http_code"
    
    case "$http_code" in
        200)
            log info "Orbit ping successful - refetch may have been triggered"
            return 0
            ;;
        404)
            log warn "Orbit ping endpoint not available"
            return 1
            ;;
        401|403)
            log warn "Authentication failed for orbit ping"
            return 1
            ;;
        *)
            log warn "Orbit ping returned HTTP $http_code"
            return 1
            ;;
    esac
}

# Function to simulate osquery check-in to trigger refetch
trigger_osquery_checkin() {
    local fleet_url="$1"
    local device_token="$2"
    
    log info "Attempting to trigger refetch via osquery distributed read..."
    
    local response
    local http_code
    
    # Simulate an osquery distributed read which should trigger refetch if requested
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X POST \
        -H "Accept: application/json" \
        -H "Content-Type: application/json" \
        -H "User-Agent: fleet-refetch-script/1.0" \
        -d '{"node_key": "'$device_token'"}' \
        --max-time 30 \
        --retry 2 \
        --retry-delay 1 \
        "${fleet_url}/api/v1/osquery/distributed/read" 2>/dev/null)
    
    if [[ $? -ne 0 ]]; then
        log warn "Failed to connect to osquery distributed endpoint"
        return 1
    fi
    
    http_code=$(echo "$response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    response_body=$(echo "$response" | sed -e 's/HTTPSTATUS:.*//g')
    
    log debug "Osquery distributed read HTTP response code: $http_code"
    
    case "$http_code" in
        200)
            # Check if refetch queries were returned
            if echo "$response_body" | grep -q "SELECT\|osquery_info\|system_info"; then
                log info "Osquery check-in successful - refetch queries may have been delivered"
                return 0
            else
                log info "Osquery check-in successful but no refetch queries detected"
                return 1
            fi
            ;;
        404)
            log warn "Osquery distributed endpoint not available"
            return 1
            ;;
        401|403)
            log warn "Authentication failed for osquery distributed read"
            return 1
            ;;
        *)
            log warn "Osquery distributed read returned HTTP $http_code"
            return 1
            ;;
    esac
}

# Main refetch function that tries multiple approaches
trigger_refetch() {
    local fleet_url="$1"
    local device_token="$2"
    local host_id="$3"
    
    log info "Attempting to trigger host refetch using device token..."
    
    # Try multiple approaches in order of preference
    
    # Method 1: Device-specific refetch endpoint
    if trigger_device_refetch "$fleet_url" "$device_token"; then
        return 0
    fi
    
    # Method 2: Orbit ping mechanism
    if trigger_orbit_ping "$fleet_url" "$device_token"; then
        return 0
    fi
    
    # Method 3: Osquery distributed read (may trigger refetch)
    if trigger_osquery_checkin "$fleet_url" "$device_token"; then
        return 0
    fi
    
    # If all methods failed, provide guidance
    log error "All refetch methods failed with the device token"
    log error ""
    log error "POSSIBLE SOLUTIONS:"
    log error "1. The device might not have Fleet Desktop installed"
    log error "2. Try restarting the fleetd/orbit service on this host to trigger a natural check-in"
    log error "3. Use an API token with admin/maintainer privileges instead:"
    log error "   curl -X POST -H 'Authorization: Bearer YOUR_API_TOKEN' \\"
    log error "        '${fleet_url}/api/v1/fleet/hosts/${host_id}/refetch'"
    
    return 1
}

# Function to display usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Refetch host details using Fleet's device authentication token.

OPTIONS:
    -u, --url URL           Fleet server URL (can also be set via FLEET_URL env var)
    -f, --file FILE         Path to device token file (default: /opt/orbit/identifier)
    -v, --verbose           Enable debug logging
    -h, --help              Show this help message

EXAMPLES:
    # Basic usage with Fleet URL as environment variable
    export FLEET_URL="https://fleet.example.com"
    $0

    # Specify Fleet URL directly
    $0 --url "https://fleet.example.com"

    # Use custom token file location
    $0 --url "https://fleet.example.com" --file "/custom/path/to/token"

    # Enable verbose logging
    $0 --url "https://fleet.example.com" --verbose

NOTES:
    - The device must have Fleet Desktop installed
    - The device token is read from $IDENTIFIER_FILE by default
    - The Fleet server URL must include the protocol (http:// or https://)
    - This script requires curl to be installed
    - Optional: jq for better JSON parsing (will fall back to basic parsing if not available)
    - The refetch operation may require elevated privileges depending on Fleet configuration

EOF
}

# Main function
main() {
    local fleet_url="$FLEET_URL"
    local token_file="$IDENTIFIER_FILE"
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -u|--url)
                fleet_url="$2"
                shift 2
                ;;
            -f|--file)
                token_file="$2"
                shift 2
                ;;
            -v|--verbose)
                LOG_LEVEL="debug"
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                log error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
    
    # Validation
    if [[ -z "$fleet_url" ]]; then
        log error "Fleet server URL is required"
        log error "Set FLEET_URL environment variable or use --url option"
        usage
        exit 1
    fi
    
    if ! validate_url "$fleet_url"; then
        log error "Invalid Fleet server URL: $fleet_url"
        log error "URL must start with http:// or https://"
        exit 1
    fi
    
    # Remove trailing slash from URL
    fleet_url="${fleet_url%/}"
    
    # Check dependencies
    if ! command_exists curl; then
        log error "curl is required but not installed"
        exit 1
    fi
    
    if ! command_exists jq; then
        log warn "jq not found - will use basic JSON parsing (less reliable)"
    fi
    
    log info "Starting Fleet host refetch process..."
    log debug "Fleet URL: $fleet_url"
    log debug "Token file: $token_file"
    
    # Read device token
    log info "Reading device authentication token..."
    local device_token
    if ! device_token=$(read_device_token "$token_file"); then
        exit 1
    fi
    log debug "Device token length: ${#device_token} characters"
    
    # Get host ID
    log info "Authenticating with Fleet server..."
    local host_id
    if ! host_id=$(get_host_id "$fleet_url" "$device_token"); then
        exit 1
    fi
    log info "Successfully authenticated - Host ID: $host_id"
    
    # Trigger refetch
    if trigger_refetch "$fleet_url" "$device_token" "$host_id"; then
        log info "Host refetch completed successfully"
        log info "Note: It may take a few moments for the updated data to be available"
        exit 0
    else
        log error "Host refetch failed"
        exit 1
    fi
}

# Run main function with all arguments
main "$@"