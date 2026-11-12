#!/usr/bin/env bash
set -euo pipefail

# backport-check.sh
#
# For each commit on PATCH_BRANCH since BASE, report whether it's INCLUDED in MINOR_BRANCH.
# INCLUDED means either:
#   1) same patch-id exists on MINOR_BRANCH (best: true cherry-pick equivalence)
#   2) normalized subject exists on MINOR_BRANCH (fallback for "Cherry pick"/"CP"/etc message variants)
#
# Output:
#   - An "Included" section with the matching MINOR SHA and match method
#   - A "Missing" section at the bottom for review
#
# Usage:
#   ./backport-check.sh <base> <patch_branch> <minor_branch>
#
# Example:
#   ./backport-check.sh fleet-v4.81.0 rc-patch-fleet-v4.81.1 rc-minor-fleet-v4.82.0

BASE="${1:?base ref required (e.g. fleet-v4.81.0)}"
PATCH_BRANCH="${2:?patch branch required}"
MINOR_BRANCH="${3:?minor branch required}"

git fetch --all --prune >/dev/null 2>&1 || true

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

normalize_subject_awk='
function norm(s) {
  # Trim leading whitespace
  gsub(/^[[:space:]]+/, "", s)

  # Strip CP / CP: (common shorthand)
  sub(/^[Cc][Pp][[:space:]]*:[[:space:]]*/, "", s)
  sub(/^[Cc][Pp][[:space:]]+/, "", s)

  # Strip Cherry-pick / Cherry pick, with or without ":", and optionally "to <branch>"
  # Examples handled:
  #   "Cherry-pick: foo"
  #   "Cherry pick: foo"
  #   "Cherry pick foo"
  #   "Cherry-pick to rc-minor... foo"
  #   "Cherry pick to rc-minor... foo"
  if (match(s, /^[Cc]herry[- ]pick([[:space:]]*:[[:space:]]*|[[:space:]]+)(to[[:space:]]+[^[:space:]]+[[:space:]]+)?/)) {
    s = substr(s, RLENGTH+1)
  }

  # Strip repeated trailing " (#12345)" blocks (PR references) at end
  while (sub(/[[:space:]]*\(#([0-9]+)\)[[:space:]]*$/, "", s)) {}

  # Collapse whitespace
  gsub(/[[:space:]]+/, " ", s)
  sub(/^[[:space:]]+/, "", s)
  sub(/[[:space:]]+$/, "", s)
  return s
}
{ print norm($0) }
'

# Reasonable start point on MINOR_BRANCH: where it diverged from BASE.
MB="$(git merge-base "$MINOR_BRANCH" "$BASE")"

echo "Indexing $MINOR_BRANCH since merge-base with $BASE ($MB)..." >&2

# Collect minor SHAs
git rev-list "$MB..$MINOR_BRANCH" > "$tmp/minor_revlist"

# Build: patch-id -> minor sha (first hit wins)
# File format: "<patchid> <sha>"
: > "$tmp/minor_patch_map"
while IFS= read -r h; do
  pid="$(
    git show "$h" --pretty=format: --patch \
      | git patch-id --stable \
      | awk '{print $1}'
  )"
  # Avoid duplicates (keep first sha we see for that patch-id)
  if ! grep -Fq "$pid " "$tmp/minor_patch_map"; then
    printf "%s %s\n" "$pid" "$h" >> "$tmp/minor_patch_map"
  fi
done < "$tmp/minor_revlist"

# Build: normalized subject -> minor sha (first hit wins)
# File format: "<normalized subject>\t<sha>"
git log --format='%H%x09%s' "$MB..$MINOR_BRANCH" \
  | while IFS=$'\t' read -r sha subj; do
      ns="$(printf "%s\n" "$subj" | awk "$normalize_subject_awk")"
      printf "%s\t%s\n" "$ns" "$sha"
    done \
  | awk -F'\t' '!seen[$1]++ { print }' \
  > "$tmp/minor_subject_map"

# Patch-branch commits to check (chronological)
git log --reverse --format='%H%x09%s' "$BASE..$PATCH_BRANCH" > "$tmp/patch_commits.tsv"

# Output buckets
included_out="$tmp/included.tsv"
missing_out="$tmp/missing.tsv"
: > "$included_out"
: > "$missing_out"

# TSV formats:
# included: status,patch_sha,minor_sha,method,subject
# missing:  status,patch_sha,subject
while IFS=$'\t' read -r sha subj; do
  norm_subj="$(printf "%s\n" "$subj" | awk "$normalize_subject_awk")"

  patchid="$(
    git show "$sha" --pretty=format: --patch \
      | git patch-id --stable \
      | awk '{print $1}'
  )"

  # 1) patch-id match
  minor_sha="$(
    grep -F "^$patchid " "$tmp/minor_patch_map" 2>/dev/null | head -n1 | awk '{print $2}'
  )" || true

  if [[ -n "${minor_sha:-}" ]]; then
    printf "INCLUDED\t%s\t%s\tpatch-id\t%s\n" "$sha" "$minor_sha" "$subj" >> "$included_out"
    continue
  fi

  # 2) normalized subject match
  minor_sha="$(
    awk -F'\t' -v k="$norm_subj" '$1==k {print $2; exit}' "$tmp/minor_subject_map" 2>/dev/null
  )" || true

  if [[ -n "${minor_sha:-}" ]]; then
    printf "INCLUDED\t%s\t%s\tsubject\t%s\n" "$sha" "$minor_sha" "$subj" >> "$included_out"
  else
    printf "MISSING\t%s\t%s\n" "$sha" "$subj" >> "$missing_out"
  fi
done < "$tmp/patch_commits.tsv"

# Pretty print
echo
echo "=== INCLUDED (present on $MINOR_BRANCH) ==="
printf "%-9s  %-12s  %-12s  %-8s  %s\n" "STATUS" "PATCH_SHA" "MINOR_SHA" "MATCH" "SUBJECT"
printf "%-9s  %-12s  %-12s  %-8s  %s\n" "--------" "------------" "------------" "--------" "----------------------------------------"

if [[ -s "$included_out" ]]; then
  while IFS=$'\t' read -r status psha msha method subject; do
    printf "%-9s  %.12s  %.12s  %-8s  %s\n" "$status" "$psha" "$msha" "$method" "$subject"
  done < "$included_out"
else
  echo "(none)"
fi

echo
echo "=== MISSING (not found on $MINOR_BRANCH) ==="
printf "%-9s  %-12s  %s\n" "STATUS" "PATCH_SHA" "SUBJECT"
printf "%-9s  %-12s  %s\n" "--------" "------------" "----------------------------------------"

if [[ -s "$missing_out" ]]; then
  while IFS=$'\t' read -r status psha subject; do
    printf "%-9s  %.12s  %s\n" "$status" "$psha" "$subject"
  done < "$missing_out"
else
  echo "(none)"
fi
