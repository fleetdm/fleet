#!/bin/bash
# Layer 1: Map REST API docs <-> handler.go routes <-> service files
# Generates OKF concept files in .okf/knowledge/api-endpoints/
#
# Parses:
#   1. server/service/handler.go — route registrations (method, path, endpointFunc)
#   2. docs/REST API/rest-api.md — endpoint documentation (heading, method, path)
#   3. server/service/*.go — endpoint function locations (file, line)
#
# Joins on normalized (HTTP method, path) to produce one concept per endpoint.

set -euo pipefail

DRY_RUN=0
if [ "${1:-}" = "--dry-run" ] || [ "${1:-}" = "-n" ]; then
    DRY_RUN=1
fi

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
HANDLER="$REPO_ROOT/server/service/handler.go"
API_DOC="$REPO_ROOT/docs/REST API/rest-api.md"
OUT_DIR="$REPO_ROOT/.okf/knowledge/api-endpoints"
SERVICE_DIR="$REPO_ROOT/server/service"

TMPDIR_WORK="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_WORK"' EXIT

if [ "$DRY_RUN" -eq 0 ]; then
    mkdir -p "$OUT_DIR"
fi

# --- Step 1: Extract routes from handler.go ---
# Output: METHOD|normalized_path|endpointFunc (sorted, deduped)
extract_routes() {
    # Captures all registrars: ue, ne, oe, mdmAppleMW, mdmAnyMW, neAppleMDM, etc.
    grep -E '\.(GET|POST|PATCH|DELETE|PUT|HEAD)\(' "$HANDLER" \
        | perl -nle 'if (/\.(GET|POST|PATCH|DELETE|PUT|HEAD)\("([^"]+)",\s*(\w+)/) { print "$1|$2|$3" }' \
        | sed 's|_version_|v1|g' \
        | sed -E 's/\{[^}]+\}/:id/g'
}

# --- Step 2: Extract endpoint docs from rest-api.md ---
# Output: METHOD|normalized_path|heading|original_path
# Only emits the first backtick endpoint line per ### heading (definition, not example)
extract_doc_endpoints() {
    local current_heading=""
    local seen_endpoint_for_heading=0
    while IFS= read -r line; do
        if echo "$line" | grep -qE '^### '; then
            current_heading="$(echo "$line" | sed 's/^### //' | sed 's/[[:space:]]*$//')"
            seen_endpoint_for_heading=0
        fi
        if [ "$seen_endpoint_for_heading" -eq 0 ] && echo "$line" | grep -qE '^\`(GET|POST|PATCH|DELETE|PUT|HEAD) /api/v1/fleet/'; then
            method="$(echo "$line" | sed -E 's/^\`(GET|POST|PATCH|DELETE|PUT|HEAD) .*/\1/')"
            path="$(echo "$line" | sed -E 's/^\`[A-Z]+ ([^\`]+)\`/\1/')"
            normalized_path="$(echo "$path" | sed -E 's/:[a-z_]+/:id/g')"
            echo "${method}|${normalized_path}|${current_heading}|${path}"
            seen_endpoint_for_heading=1
        fi
    done < "$API_DOC"
}

# --- Step 3: Find which .go file defines each endpoint function ---
find_endpoint_file() {
    local func_name="$1"
    grep -rn "func ${func_name}(" "$SERVICE_DIR"/*.go 2>/dev/null \
        | head -1 \
        | sed -E 's/^.*\/(server\/service\/[^:]+):([0-9]+):.*/\1:\2/' \
        || true
}

# --- Step 4: Build lookup files and join ---

echo "Extracting routes from handler.go..."
extract_routes | sort -t'|' -k1,2 -u > "$TMPDIR_WORK/routes.tsv"
route_count="$(wc -l < "$TMPDIR_WORK/routes.tsv" | tr -d ' ')"
echo "  Found $route_count routes"

echo "Extracting endpoints from rest-api.md..."
extract_doc_endpoints | sort -t'|' -k1,2 -u > "$TMPDIR_WORK/docs.tsv"
doc_count="$(wc -l < "$TMPDIR_WORK/docs.tsv" | tr -d ' ')"
echo "  Found $doc_count documented endpoints"

echo "Joining routes with docs..."
matched=0
code_only=0

while IFS='|' read -r method norm_path func; do
    key="${method}|${norm_path}"

    # Look up doc info
    doc_line="$(grep -F "$key" "$TMPDIR_WORK/docs.tsv" | head -1 || true)"
    if [ -n "$doc_line" ]; then
        matched=$((matched + 1))
    else
        code_only=$((code_only + 1))
    fi
done < "$TMPDIR_WORK/routes.tsv"

# Count doc-only endpoints (documented but no code route)
doc_only=0
while IFS='|' read -r method norm_path heading orig_path; do
    key="${method}|${norm_path}"
    if ! grep -qF "$key" "$TMPDIR_WORK/routes.tsv"; then
        doc_only=$((doc_only + 1))
    fi
done < "$TMPDIR_WORK/docs.tsv"

total_to_write=$((matched + code_only))

echo ""
echo "=== Layer 1 Results ==="
echo "Matched (code + docs):  $matched"
echo "Code only (no docs):    $code_only"
echo "Docs only (no code):    $doc_only"
echo "Files to write:         $total_to_write"
echo "Output directory:       $OUT_DIR"

if [ "$DRY_RUN" -eq 1 ]; then
    echo ""
    echo "[DRY RUN] No files written. Run without --dry-run to generate concept files."
    echo ""
    echo "Sample matches:"
    head -5 "$TMPDIR_WORK/routes.tsv" | while IFS='|' read -r method norm_path func; do
        doc_line="$(grep -F "${method}|${norm_path}" "$TMPDIR_WORK/docs.tsv" | head -1 || true)"
        if [ -n "$doc_line" ]; then
            heading="$(echo "$doc_line" | cut -d'|' -f3)"
            echo "  ✓ $method $norm_path → $func → doc: \"$heading\""
        else
            echo "  ✗ $method $norm_path → $func → (no docs)"
        fi
    done
    exit 0
fi

# --- Generate concept files ---
echo ""
echo "Writing concept files..."

while IFS='|' read -r method norm_path func; do
    key="${method}|${norm_path}"

    # Look up doc info
    doc_line="$(grep -F "$key" "$TMPDIR_WORK/docs.tsv" | head -1 || true)"
    heading=""
    orig_path="$norm_path"
    if [ -n "$doc_line" ]; then
        heading="$(echo "$doc_line" | cut -d'|' -f3)"
        orig_path="$(echo "$doc_line" | cut -d'|' -f4)"
    else
        heading="$func"
    fi

    # Find service file location
    location="$(find_endpoint_file "$func")"
    service_file=""
    service_line=""
    if [ -n "$location" ]; then
        service_file="${location%%:*}"
        service_line="${location##*:}"
    fi

    # Generate slug for filename
    slug="$(echo "${method}-${orig_path}" | tr '[:upper:]' '[:lower:]' | sed -E 's|/api/v1/fleet/||' | sed 's|[/:{}]|-|g' | sed 's|--*|-|g' | sed 's|-$||')"

    # Lowercase method for tag
    method_lower="$(echo "$method" | tr '[:upper:]' '[:lower:]')"

    # Build concept file
    cat > "$OUT_DIR/${slug}.md" <<CONCEPT_EOF
---
type: API Endpoint
title: "${heading}"
description: "${method} ${orig_path}"
resource: "${orig_path}"
tags:
    - api
    - ${method_lower}
    - generated
    - layer1
generated: true
generator: tools/okf/layer1-api-mapping.sh
---

## ${heading}

\`${method} ${orig_path}\`

### Code

- **Endpoint function:** \`${func}\`
CONCEPT_EOF

    if [ -n "$service_file" ]; then
        cat >> "$OUT_DIR/${slug}.md" <<CONCEPT_EOF
- **Service file:** [${service_file}](../../${service_file})
- **Line:** ${service_line}
CONCEPT_EOF
    fi

    if [ -n "$doc_line" ]; then
        anchor="$(echo "$heading" | tr '[:upper:]' '[:lower:]' | sed 's/ /-/g' | sed 's/[^a-z0-9-]//g')"
        cat >> "$OUT_DIR/${slug}.md" <<CONCEPT_EOF

### Documentation

- [REST API docs](../../docs/REST%20API/rest-api.md#${anchor})
CONCEPT_EOF
    fi

done < "$TMPDIR_WORK/routes.tsv"

# --- Generate index.md ---
cat > "$OUT_DIR/index.md" <<INDEX_HEADER
---
type: index
title: API Endpoints
description: Index of all Fleet REST API endpoints mapped to service code
tags:
    - api
    - index
    - generated
    - layer1
generated: true
generator: tools/okf/layer1-api-mapping.sh
---

# API Endpoints

${matched} endpoints matched to docs, ${code_only} code-only (undocumented).

| Method | Path | Title | Service File |
|--------|------|-------|-------------|
INDEX_HEADER

while IFS='|' read -r method norm_path func; do
    key="${method}|${norm_path}"
    doc_line="$(grep -F "$key" "$TMPDIR_WORK/docs.tsv" | head -1 || true)"
    heading=""
    orig_path="$norm_path"
    if [ -n "$doc_line" ]; then
        heading="$(echo "$doc_line" | cut -d'|' -f3)"
        orig_path="$(echo "$doc_line" | cut -d'|' -f4)"
    else
        heading="$func"
    fi
    slug="$(echo "${method}-${orig_path}" | tr '[:upper:]' '[:lower:]' | sed -E 's|/api/v1/fleet/||' | sed 's|[/:{}]|-|g' | sed 's|--*|-|g' | sed 's|-$||')"

    location="$(find_endpoint_file "$func")"
    service_file=""
    if [ -n "$location" ]; then
        service_file="${location%%:*}"
    fi

    echo "| ${method} | \`${orig_path}\` | [${heading}](${slug}.md) | ${service_file} |" >> "$OUT_DIR/index.md"
done < "$TMPDIR_WORK/routes.tsv"

echo "Done. Wrote $total_to_write concept files + index.md to $OUT_DIR"
