#!/bin/bash

# Partition Windows FMA slugs by installer architecture and emit a GitHub
# Actions job matrix that routes each app to a runner with the matching native
# architecture: arm64 apps to windows-11-arm, everything else (x64, x86,
# neutral) to the x64 runner. Buckets larger than the shard size are split
# into shards that validate in parallel.
#
# The architecture for a slug like "7-zip/windows" is read from
# ee/maintained-apps/inputs/winget/7-zip.json (.installer_arch). Slugs whose
# input file or installer_arch is missing default to the x64 runner.
#
# Usage: partition-fma-windows-by-arch.sh <slugs_json_array | slugs_json_file> [shard_size]
#
# Like filter-apps-json.sh, the slugs argument is either a literal JSON array
# string or a path to a file containing one. Non-windows slugs are ignored.
#
# Outputs (appended to $GITHUB_OUTPUT):
#   has_windows_apps - "true" or "false"
#   matrix           - JSON array of {name, runner, arch, slugs} objects, where
#                      slugs is a JSON-encoded array string for that shard.

set -euo pipefail

X64_RUNNER="windows-latest"
ARM64_RUNNER="windows-11-arm"

REPO_ROOT="${GITHUB_WORKSPACE:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
INPUTS_DIR="${REPO_ROOT}/ee/maintained-apps/inputs/winget"
GITHUB_OUTPUT="${GITHUB_OUTPUT:-/dev/stdout}"

if ! command -v jq &> /dev/null; then
    echo "Error: jq is required but not installed" >&2
    exit 1
fi

SLUGS_INPUT="${1:-[]}"
SHARD_SIZE="${2:-25}"

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

WINDOWS_SLUGS_JSON=$(jq -c '[.[] | select(endswith("/windows"))] | unique' <<< "$SLUGS_JSON")
TOTAL=$(jq 'length' <<< "$WINDOWS_SLUGS_JSON")

if [ "$TOTAL" -eq 0 ]; then
    echo "No Windows apps to validate."
    echo "has_windows_apps=false" >> "$GITHUB_OUTPUT"
    echo "matrix=[]" >> "$GITHUB_OUTPUT"
    exit 0
fi

x64_slugs=()
arm64_slugs=()
while IFS= read -r slug; do
    [ -z "$slug" ] && continue
    name="${slug%/windows}"
    input_file="${INPUTS_DIR}/${name}.json"
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
done < <(jq -r '.[]' <<< "$WINDOWS_SLUGS_JSON")

ENTRIES_FILE="$(mktemp)"
trap 'rm -f "$ENTRIES_FILE"' EXIT

# emit_shards <arch> <runner> <slug...>
emit_shards() {
    local arch="$1" runner="$2"
    shift 2
    local slugs=("$@")
    local total=${#slugs[@]}
    [ "$total" -eq 0 ] && return 0
    local shards=$(( (total + SHARD_SIZE - 1) / SHARD_SIZE ))
    local i=0 shard=1
    while [ "$i" -lt "$total" ]; do
        local chunk=("${slugs[@]:$i:$SHARD_SIZE}")
        local chunk_json
        chunk_json=$(printf '%s\n' "${chunk[@]}" | jq -R . | jq -s -c .)
        local name="$arch"
        if [ "$shards" -gt 1 ]; then
            name="$arch (${shard}/${shards})"
        fi
        jq -c -n --arg name "$name" --arg runner "$runner" --arg arch "$arch" --arg slugs "$chunk_json" \
            '{name: $name, runner: $runner, arch: $arch, slugs: $slugs}' >> "$ENTRIES_FILE"
        i=$((i + SHARD_SIZE))
        shard=$((shard + 1))
    done
}

emit_shards "x64" "$X64_RUNNER" ${x64_slugs[@]+"${x64_slugs[@]}"}
emit_shards "arm64" "$ARM64_RUNNER" ${arm64_slugs[@]+"${arm64_slugs[@]}"}

MATRIX_JSON=$(jq -c -s . "$ENTRIES_FILE")

echo "Windows apps to validate: $TOTAL (x64/x86/neutral: ${#x64_slugs[@]}, arm64: ${#arm64_slugs[@]})"
echo "Matrix: $MATRIX_JSON"

echo "has_windows_apps=true" >> "$GITHUB_OUTPUT"
echo "matrix=${MATRIX_JSON}" >> "$GITHUB_OUTPUT"
