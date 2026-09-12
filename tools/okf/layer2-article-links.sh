#!/bin/bash
# Layer 2: Extract outbound links from articles to docs and guides
# Generates OKF concept files in .okf/knowledge/article-links/
#
# Scans articles/*.md for links matching:
#   - fleetdm.com/docs/...  → maps to docs/ directory
#   - fleetdm.com/guides/... → maps to articles/ directory (guides are articles with category=guides)
#
# Produces one concept file per article that has outbound links,
# plus an index.md for progressive disclosure.

set -euo pipefail

DRY_RUN=0
if [ "${1:-}" = "--dry-run" ] || [ "${1:-}" = "-n" ]; then
    DRY_RUN=1
fi

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
ARTICLES_DIR="$REPO_ROOT/articles"
DOCS_DIR="$REPO_ROOT/docs"
OUT_DIR="$REPO_ROOT/.okf/knowledge/article-links"

TMPDIR_WORK="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_WORK"' EXIT

# --- Step 1: For each article, extract outbound doc/guide links ---
# Output: article_file|link_type|url|local_path
extract_links() {
    for article in "$ARTICLES_DIR"/*.md; do
        filename="$(basename "$article" .md)"

        # Extract fleetdm.com/docs/ and /guides/ links
        { grep -oE 'https://fleetdm\.com/(docs|guides)/[^ )"'"'"']*' "$article" 2>/dev/null || true; } | sort -u | while read -r url; do
            # Strip trailing punctuation and anchors for file resolution
            clean_url="$(echo "$url" | sed 's/#.*//' | sed 's/[.,;:)]*$//')"

            if echo "$url" | grep -q 'fleetdm\.com/guides/'; then
                # guides/slug → articles/slug.md
                slug="$(echo "$clean_url" | sed 's|.*/guides/||')"
                local_path="articles/${slug}.md"
                link_type="guide"
            else
                # docs/section/slug → docs/Section/slug.md (case-insensitive lookup)
                url_path="$(echo "$clean_url" | sed 's|.*/docs/||')"
                # Try to find the file case-insensitively
                local_path="$(find "$DOCS_DIR" -ipath "*/${url_path}.md" 2>/dev/null | head -1 | sed "s|$REPO_ROOT/||")"
                if [ -z "$local_path" ]; then
                    # Try without the section prefix (e.g., docs/rest-api/rest-api → docs/REST API/rest-api.md)
                    slug="$(basename "$url_path")"
                    local_path="$(find "$DOCS_DIR" -iname "${slug}.md" 2>/dev/null | head -1 | sed "s|$REPO_ROOT/||")"
                fi
                link_type="doc"
            fi

            echo "${filename}|${link_type}|${url}|${local_path}"
        done
    done
}

echo "Scanning articles for doc/guide links..."
extract_links > "$TMPDIR_WORK/all_links.tsv"

total_links="$(wc -l < "$TMPDIR_WORK/all_links.tsv" | tr -d ' ')"
articles_with_links="$(cut -d'|' -f1 "$TMPDIR_WORK/all_links.tsv" | sort -u | wc -l | tr -d ' ')"
doc_links="$(grep '|doc|' "$TMPDIR_WORK/all_links.tsv" | wc -l | tr -d ' ')"
guide_links="$(grep '|guide|' "$TMPDIR_WORK/all_links.tsv" | wc -l | tr -d ' ')"
resolved="$(awk -F'|' '$4 != ""' "$TMPDIR_WORK/all_links.tsv" | wc -l | tr -d ' ')"
unresolved="$(awk -F'|' '$4 == ""' "$TMPDIR_WORK/all_links.tsv" | wc -l | tr -d ' ')"

echo ""
echo "=== Layer 2 Results ==="
echo "Articles with links:    $articles_with_links"
echo "Total links found:      $total_links"
echo "  Doc links:            $doc_links"
echo "  Guide links:          $guide_links"
echo "Resolved to local file: $resolved"
echo "Unresolved:             $unresolved"
echo "Files to write:         $articles_with_links (+ index.md)"
echo "Output directory:       $OUT_DIR"

if [ "$DRY_RUN" -eq 1 ]; then
    echo ""
    echo "[DRY RUN] No files written. Run without --dry-run to generate concept files."
    echo ""
    echo "Sample (first 10 articles with links):"
    cut -d'|' -f1 "$TMPDIR_WORK/all_links.tsv" | sort -u | head -10 | while read -r article; do
        count="$(grep -c "^${article}|" "$TMPDIR_WORK/all_links.tsv")"
        echo "  ${article}.md → ${count} links"
    done
    echo ""
    echo "Sample unresolved links:"
    awk -F'|' '$4 == ""' "$TMPDIR_WORK/all_links.tsv" | head -5 | while IFS='|' read -r article type url path; do
        echo "  ${article}: ${url}"
    done
    exit 0
fi

# --- Generate concept files ---
echo ""
echo "Writing concept files..."
mkdir -p "$OUT_DIR"

cut -d'|' -f1 "$TMPDIR_WORK/all_links.tsv" | sort -u | while read -r article; do
    # Get title from first heading in the article
    title="$(head -5 "$ARTICLES_DIR/${article}.md" | grep -m1 '^# ' | sed 's/^# //' || echo "$article")"
    if [ -z "$title" ]; then
        title="$article"
    fi

    cat > "$OUT_DIR/${article}.md" <<CONCEPT_EOF
---
type: Article
title: "${title}"
description: "Links from articles/${article}.md to docs and guides"
resource: "articles/${article}.md"
tags:
    - article
    - generated
    - layer2
generated: true
generator: tools/okf/layer2-article-links.sh
---

## ${title}

Source: [articles/${article}.md](../../articles/${article}.md)

### Links to docs and guides

CONCEPT_EOF

    grep "^${article}|" "$TMPDIR_WORK/all_links.tsv" | while IFS='|' read -r _ link_type url local_path; do
        if [ -n "$local_path" ]; then
            echo "- [${url}](../../${local_path})" >> "$OUT_DIR/${article}.md"
        else
            echo "- ${url} *(unresolved)*" >> "$OUT_DIR/${article}.md"
        fi
    done
done

# --- Generate index.md ---
cat > "$OUT_DIR/index.md" <<INDEX_HEADER
---
type: index
title: Article Links
description: Index of Fleet articles and their outbound links to docs and guides
tags:
    - article
    - index
    - generated
    - layer2
generated: true
generator: tools/okf/layer2-article-links.sh
---

# Article Links

${articles_with_links} articles link to docs or guides. ${total_links} total links (${doc_links} to docs, ${guide_links} to guides).

| Article | Doc Links | Guide Links | Total |
|---------|-----------|-------------|-------|
INDEX_HEADER

cut -d'|' -f1 "$TMPDIR_WORK/all_links.tsv" | sort -u | while read -r article; do
    d="$(grep -c "^${article}|doc|" "$TMPDIR_WORK/all_links.tsv" 2>/dev/null || echo 0)"
    g="$(grep -c "^${article}|guide|" "$TMPDIR_WORK/all_links.tsv" 2>/dev/null || echo 0)"
    t=$((d + g))
    echo "| [${article}](${article}.md) | ${d} | ${g} | ${t} |" >> "$OUT_DIR/index.md"
done

echo "Done. Wrote $articles_with_links concept files + index.md to $OUT_DIR"
