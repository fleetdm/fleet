#!/bin/bash

# Partition FMA slugs for a platform into a GitHub Actions job matrix whose
# entries route apps to an appropriate runner, splitting buckets larger than
# the shard size into shards that validate in parallel.
#
#   windows  Apps are routed to a runner with the matching native installer
#            architecture: arm64 apps to windows-11-arm, everything else
#            (x64, x86, neutral) to the x64 runner. The architecture for a
#            slug like "7-zip/windows" is read from
#            ee/maintained-apps/inputs/winget/7-zip.json (.installer_arch).
#            Slugs whose input file or installer_arch is missing default to
#            the x64 runner.
#   darwin   All apps run on macos-latest (arm64; x86-only casks run under
#            Rosetta 2, matching how customer Macs run them). No architecture
#            partitioning is needed.
#
# Usage: partition-fma-apps.sh <windows|darwin> <slugs_json_array | slugs_json_file> [shard_size]
#
# Like filter-apps-json.sh, the slugs argument is either a literal JSON array
# string or a path to a file containing one. Slugs for other platforms are
# ignored.
#
# Outputs (appended to $GITHUB_OUTPUT):
#   has_apps - "true" or "false"
#   matrix   - JSON array of {name, runner, slugs} objects, where slugs is a
#              JSON-encoded array string for that shard. Windows entries also
#              carry an "arch" field.

set -euo pipefail

WINDOWS_X64_RUNNER="windows-latest"
WINDOWS_ARM64_RUNNER="windows-11-arm"
DARWIN_RUNNER="macos-latest"

REPO_ROOT="${GITHUB_WORKSPACE:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
WINGET_INPUTS_DIR="${REPO_ROOT}/ee/maintained-apps/inputs/winget"
GITHUB_OUTPUT="${GITHUB_OUTPUT:-/dev/stdout}"

if ! command -v jq &> /dev/null; then
    echo "Error: jq is required but not installed" >&2
    exit 1
fi

PLATFORM="${1:-}"
SLUGS_INPUT="${2:-[]}"
SHARD_SIZE="${3:-25}"

if [ "$PLATFORM" != "windows" ] && [ "$PLATFORM" != "darwin" ]; then
    echo "Error: platform must be 'windows' or 'darwin', got '$PLATFORM'" >&2
    exit 1
fi

if [ -n "$SLUGS_INPUT" ] && [ -f "$SLUGS_INPUT" ]; then
    SLUGS_JSON="$(cat "$SLUGS_INPUT")"
else
    SLUGS_JSON="$SLUGS_INPUT"
fi
if [ -z "$SLUGS_JSON" ] || [ "$SLUGS_JSON" == "null" ]; then
    SLUGS_JSON="[]"
fi

if ! [[ "$SHARD_SIZE" =~ ^[1-9][0-9]*$ ]]; then
    echo "Error: shard size must be a positive integer, got '$SHARD_SIZE'" >&2
    exit 1
fi

PLATFORM_SLUGS_JSON=$(jq -c --arg suffix "/${PLATFORM}" '[.[] | select(endswith($suffix))] | unique' <<< "$SLUGS_JSON")
TOTAL=$(jq 'length' <<< "$PLATFORM_SLUGS_JSON")

if [ "$TOTAL" -eq 0 ]; then
    echo "No ${PLATFORM} apps to validate."
    echo "has_apps=false" >> "$GITHUB_OUTPUT"
    echo "matrix=[]" >> "$GITHUB_OUTPUT"
    exit 0
fi

ENTRIES_FILE="$(mktemp)"
trap 'rm -f "$ENTRIES_FILE"' EXIT

# emit_shards <bucket_name> <runner> <arch> <slug...>
# arch is embedded in the matrix entry when non-empty (windows only).
emit_shards() {
    local bucket="$1" runner="$2" arch="$3"
    shift 3
    local slugs=("$@")
    local total=${#slugs[@]}
    [ "$total" -eq 0 ] && return 0
    local shards=$(( (total + SHARD_SIZE - 1) / SHARD_SIZE ))
    local i=0 shard=1
    while [ "$i" -lt "$total" ]; do
        local chunk=("${slugs[@]:$i:$SHARD_SIZE}")
        local chunk_json
        chunk_json=$(printf '%s\n' "${chunk[@]}" | jq -R . | jq -s -c .)
        local name="$bucket"
        if [ "$shards" -gt 1 ]; then
            name="$bucket (${shard}/${shards})"
        fi
        jq -c -n --arg name "$name" --arg runner "$runner" --arg arch "$arch" --arg slugs "$chunk_json" \
            '{name: $name, runner: $runner, slugs: $slugs} + (if $arch != "" then {arch: $arch} else {} end)' >> "$ENTRIES_FILE"
        i=$((i + SHARD_SIZE))
        shard=$((shard + 1))
    done
}

case "$PLATFORM" in
    windows)
        x64_slugs=()
        arm64_slugs=()
        while IFS= read -r slug; do
            [ -z "$slug" ] && continue
            name="${slug%/windows}"
            input_file="${WINGET_INPUTS_DIR}/${name}.json"
            arch=""
            if [ -f "$input_file" ]; then
                arch=$(jq -r '.installer_arch // empty' "$input_file" 2>/dev/null || echo "")
            else
                echo "Warning: no winget input file for '$slug' at $input_file, assuming x64" >&2
            fi
            case "$arch" in
                arm64)
                    arm64_slugs+=("$slug")
                    ;;
                *)
                    # x64, x86 and neutral installers all run natively on the x64 runner.
                    x64_slugs+=("$slug")
                    ;;
            esac
            echo "  - $slug -> ${arch:-x64}"
        done < <(jq -r '.[]' <<< "$PLATFORM_SLUGS_JSON")

        emit_shards "x64" "$WINDOWS_X64_RUNNER" "x64" ${x64_slugs[@]+"${x64_slugs[@]}"}
        emit_shards "arm64" "$WINDOWS_ARM64_RUNNER" "arm64" ${arm64_slugs[@]+"${arm64_slugs[@]}"}

        echo "Windows apps to validate: $TOTAL (x64/x86/neutral: ${#x64_slugs[@]}, arm64: ${#arm64_slugs[@]})"
        ;;
    darwin)
        darwin_slugs=()
        while IFS= read -r slug; do
            [ -z "$slug" ] && continue
            darwin_slugs+=("$slug")
            echo "  - $slug"
        done < <(jq -r '.[]' <<< "$PLATFORM_SLUGS_JSON")

        emit_shards "darwin" "$DARWIN_RUNNER" "" ${darwin_slugs[@]+"${darwin_slugs[@]}"}

        echo "Darwin apps to validate: $TOTAL"
        ;;
esac

MATRIX_JSON=$(jq -c -s . "$ENTRIES_FILE")
echo "Matrix: $MATRIX_JSON"

echo "has_apps=true" >> "$GITHUB_OUTPUT"
echo "matrix=${MATRIX_JSON}" >> "$GITHUB_OUTPUT"
