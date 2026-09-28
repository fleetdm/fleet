#!/usr/bin/env bash
# Checks handbook changes on the current branch for problems that break the
# fleetdm.com build, fail CI, or break links people already use.
#
# Usage (from anywhere in the repo):
#   bash .claude/skills/handbook-pr/scripts/check-handbook-diff.sh [base-ref]
#
# base-ref defaults to origin/main. The working tree (committed, uncommitted,
# and untracked handbook files) is compared against the merge base.
# Exits 1 if any errors were found. Warnings don't change the exit code.

set -uo pipefail

base_ref="${1:-origin/main}"
repo_root=$(git rev-parse --show-toplevel) || exit 2
cd "$repo_root" || exit 2

if ! base=$(git merge-base HEAD "$base_ref" 2>/dev/null); then
  echo "Couldn't find a merge base with $base_ref. Run 'git fetch origin main' and try again." >&2
  exit 2
fi

# Keep non-ASCII paths (like handbook/company/legal/📜 ...) unquoted in git output.
g() { git -c core.quotePath=false "$@"; }

errors=0
warnings=0
err() { echo "ERROR  $1"; errors=$((errors + 1)); }
warn() { echo "WARN   $1"; warnings=$((warnings + 1)); }

# Mirrors how website/api/helpers/strings/to-html.js builds heading IDs:
# _.kebabCase (lodash 3) of the lowercased heading HTML, with apostrophes removed.
slugify() {
  printf '%s' "$1" |
    sed -E 's/^(>[[:space:]]*)?#+[[:space:]]+//; s/[[:space:]]+#*[[:space:]]*$//' |
    sed -E 's/`([^`]*)`/ code \1 code /g; s/\*\*([^*]*)\*\*/ strong \1 strong /g; s/\*([^*]+)\*/ em \1 em /g' |
    sed -E 's/(^|[^[:alnum:]])_([^_]+)_([^[:alnum:]]|$)/\1 em \2 em \3/g' |
    sed -e "s/’//g" -e "s/'//g" |
    tr '[:upper:]' '[:lower:]' |
    LC_ALL=C grep -oE '[a-z]+|[0-9]+' |
    paste -sd- -
}

# The fleetdm.com path for a handbook file: handbook/it/README.md -> handbook/it
url_path() {
  local p="${1%.md}"
  p="${p%/README}"
  printf '%s' "$p"
}

utf16_length() {
  printf '%s' "$1" | iconv -f UTF-8 -t UTF-16LE 2>/dev/null | wc -c | awk '{print $1 / 2}'
}

untracked=$(g ls-files --others --exclude-standard -- handbook/)
changed=$(g diff --name-only --diff-filter=d "$base" -- handbook/; printf '%s\n' "$untracked")
changed=$(printf '%s\n' "$changed" | sed '/^$/d' | sort -u)

if [ -z "$changed" ] && [ -z "$(g diff --name-only --diff-filter=D "$base" -- handbook/)" ]; then
  echo "No handbook changes found against $base_ref."
  exit 0
fi

outside=$( { g diff --name-only "$base"; g ls-files --others --exclude-standard; } | grep -v '^handbook/' | sort -u)
if [ -n "$outside" ]; then
  warn "This branch also changes files outside handbook/. Keep handbook PRs handbook-only, or explain why in the PR description:"
  printf '%s\n' "$outside" | sed 's/^/         /'
fi

# Deleted or moved pages change a fleetdm.com URL.
while IFS=$'\t' read -r status old new; do
  [ -z "$status" ] && continue
  case "$status" in
    D) [[ "$old" == *.md ]] && warn "$old was deleted. Links to https://fleetdm.com/$(url_path "$old") will 404. Search the repo for that path and update links." ;;
    R*) [[ "$old" == *.md ]] && warn "$old was moved to $new. Links to https://fleetdm.com/$(url_path "$old") will 404. Search the repo for that path and update links." ;;
  esac
done < <(g diff --name-status -M "$base" -- handbook/)

# Added and removed lines, one record per line: kind<TAB>file<TAB>line<TAB>text
records=$(
  {
    g diff --unified=0 --no-color --no-ext-diff "$base" -- handbook/
    while IFS= read -r f; do
      [ -n "$f" ] && g diff --no-index --unified=0 --no-color /dev/null "$f"
    done <<< "$untracked"
  } 2>/dev/null | awk '
    /^diff --git / { hdr = 1; file = ""; next }
    hdr && /^\+\+\+ / { f = substr($0, 5); if (f == "/dev/null") file = ""; else { sub(/^b\//, "", f); file = f }; next }
    /^@@ / { hdr = 0; match($0, /\+[0-9]+/); ln = substr($0, RSTART + 1, RLENGTH - 1) + 0; next }
    hdr || file == "" { next }
    /^\+/ { printf "A\t%s\t%d\t%s\n", file, ln, substr($0, 2); ln++; next }
    /^-/ { printf "R\t%s\t0\t%s\n", file, substr($0, 2); next }
  '
)

re_heading='^(>[[:space:]]*)?#{1,6}[[:space:]]'
re_here='\[(click )?here\]'
re_fleetdm='(^|[^a-zA-Z./@-])(FleetDM|fleetDM)([^a-zA-Z.]|$)'
re_maintainer='<meta name="maintainedBy" value="([A-Za-z0-9-]+)"'
re_dri='^[-[:space:]]*dri:[[:space:]]*"?([A-Za-z0-9-]+)"?'

removed_headings=""
added_headings=""
new_handles=""

while IFS=$'\t' read -r kind file ln text; do
  [ -z "$kind" ] && continue
  if [[ "$file" == *.md ]]; then
    if [[ "$text" =~ $re_heading ]]; then
      slug=$(slugify "$text")
      if [ "$kind" = "R" ]; then
        removed_headings+="$file"$'\t'"$slug"$'\t'"$text"$'\n'
      else
        added_headings+="$file"$'\t'"$slug"$'\n'
      fi
    fi
    [ "$kind" = "A" ] || continue
    loc="$file:$ln"
    [[ "$text" == *"@fleetdm.com"* ]] && err "$loc has an @fleetdm.com email address, which fails the website build. Use @example.com, or link to a Slack channel or handbook section."
    shopt -s nocasematch
    [[ "$text" =~ $re_here ]] && err "$loc uses \"here\" as link text, which fails the docs CI check. Link descriptive words instead."
    shopt -u nocasematch
    if [[ "$text" == *"{{"*"}}"* ]]; then
      [[ "$(printf '%s' "$text" | sed 's/`[^`]*`//g')" == *"{{"*"}}"* ]] &&
        warn "$loc has {{ }} outside inline code, which can fail the website build. Wrap it in backticks."
    fi
    [[ "$text" == *"—"* ]] && warn "$loc has an em dash. writing.md prefers a comma, a colon, or a new sentence."
    [[ "$text" == *"](#"* ]] && warn "$loc links with a relative anchor. Use the full https://fleetdm.com/$(url_path "$file")#... URL."
    [[ "$text" == *"](./"* || "$text" == *"](../"* ]] && warn "$loc uses a relative path in a link or image. Use a full https:// URL."
    [[ "$text" =~ $re_fleetdm ]] && warn "$loc says \"${BASH_REMATCH[2]}\". Use \"Fleet\" in prose."
    if [[ "$text" =~ $re_maintainer ]]; then
      new_handles+="${BASH_REMATCH[1]}"$'\t'"$loc"$'\n'
    fi
  elif [[ "$file" == *.rituals.yml && "$kind" = "A" ]]; then
    if [[ "$text" =~ $re_dri ]]; then
      new_handles+="${BASH_REMATCH[1]}"$'\t'"$file:$ln"$'\n'
    fi
  fi
done <<< "$records"

# Removed or renamed headings break anchors. A heading that only changed level keeps its slug.
while IFS=$'\t' read -r file slug text; do
  [ -z "$file" ] && continue
  [ -z "$slug" ] && continue
  printf '%s' "$added_headings" | grep -qxF "$file"$'\t'"$slug" && continue
  page=$(url_path "$file")
  warn "$file: heading \"$text\" was renamed or removed, so https://fleetdm.com/$page#$slug no longer resolves. Update these links in this PR (links in Slack and GitHub issues can't be checked):"
  matches=$(
    g grep -n -F -e "$page#$slug" -e "$(basename "$file")#$slug" -- . ':!node_modules' 2>/dev/null
    g grep -n -F -e "(#$slug)" -- "$file" 2>/dev/null
  )
  if [ -n "$matches" ]; then
    printf '%s\n' "$matches" | sort -u | sed 's/^/         /'
  else
    echo "         (no links found in the repo)"
  fi
done <<< "$removed_headings"

# Page-level rules the website build enforces for every handbook Markdown page.
while IFS= read -r file; do
  [[ "$file" == *.md && -f "$file" ]] || continue
  grep -qE '<meta name="maintainedBy" value="[^"]+"' "$file" ||
    err "$file is missing <meta name=\"maintainedBy\" value=\"<github-username>\">, which fails the website build. Copy it from the parent page."
  title=$(sed -nE 's/.*<meta name="title" value="([^"]*)".*/\1/p' "$file" | head -1)
  if [ -n "$title" ]; then
    len=$(utf16_length "$title")
    [ "${len%.*}" -gt 40 ] && err "$file has a title meta tag longer than 40 characters (\"$title\" is $len; emoji count as 2), which fails the website build."
  fi
  if ! git cat-file -e "$base:$file" 2>/dev/null; then
    name=$(basename "$file")
    [[ "$name" == "README.md" || "$name" =~ ^[a-z0-9]+(-[a-z0-9]+)*\.md$ ]] ||
      warn "$file: new page filenames should be the kebab-case, lowercase page title (for example why-this-way.md)."
    grep -qE '^# ' "$file" || warn "$file: new pages need an H1 (# Page title) that matches the page name."
  fi
done <<< "$changed"

# Rituals: fields the website build requires, plus the autoIssue frequency rule.
yaml_check() {
  if python3 -c 'import yaml' 2>/dev/null; then
    python3 -c 'import sys, yaml
try:
    yaml.safe_load(open(sys.argv[1]))
except Exception as e:
    print(" ".join(str(e).split()))' "$1"
  elif command -v ruby >/dev/null 2>&1; then
    ruby -ryaml -e 'begin; YAML.load_file(ARGV[0]); rescue StandardError, Psych::SyntaxError => e; puts e.message.split.join(" "); end' "$1"
  else
    echo "SKIPPED"
  fi
}
while IFS= read -r file; do
  [[ "$file" == *.rituals.yml && -f "$file" ]] || continue
  parse_result=$(yaml_check "$file")
  if [ "$parse_result" = "SKIPPED" ]; then
    echo "NOTE   $file: no YAML parser found (python3 with PyYAML, or ruby), so YAML syntax wasn't checked. CI will check it."
  elif [ -n "$parse_result" ]; then
    err "$file isn't valid YAML: $parse_result"
    continue
  fi
  while IFS= read -r problem; do
    [ -n "$problem" ] && err "$problem"
  done < <(awk -v f="$file" '
    function unquote(v) { sub(/[ \t]+#.*$/, "", v); sub(/^"/, "", v); sub(/"[ \t]*$/, "", v); return v }
    function parse(l,   key, val) {
      key = l; sub(/:.*/, "", key)
      val = l; sub(/^[^:]*:[ \t]*/, "", val); val = unquote(val)
      if (key == "autoIssue") { auto = 1; seen[key] = 1; return }
      if (val != "") seen[key] = 1
      if (key == "frequency") freq = val
      if (key == "task") task = val
    }
    function flush(   i) {
      if (!in_item) return
      for (i = 1; i <= nreq; i++) if (!(req[i] in seen)) printf "%s:%d: ritual \"%s\" is missing %s, which fails the website build.\n", f, start, task, req[i]
      if (auto && !(freq in ok)) printf "%s:%d: ritual \"%s\" has autoIssue, so frequency must be Daily, Weekly, Triweekly, Monthly, Quarterly, or Annually (found \"%s\").\n", f, start, task, freq
      split("", seen); auto = 0; freq = ""; task = ""
    }
    BEGIN {
      nreq = split("task startedOn frequency description dri", req, " ")
      split("Daily Weekly Triweekly Monthly Quarterly Annually", o, " "); for (i in o) ok[o[i]] = 1
    }
    /^-/ { flush(); in_item = 1; start = NR; l = $0; sub(/^-[ \t]*/, "", l); if (l ~ /^[A-Za-z]+:/) parse(l); next }
    in_item && /^  [A-Za-z]+:/ { l = $0; sub(/^  /, "", l); parse(l); next }
    END { flush() }
  ' "$file")
done <<< "$changed"

# GitHub usernames added in maintainedBy or dri must exist.
if [ -n "$new_handles" ]; then
  if gh auth status >/dev/null 2>&1; then
    while IFS=$'\t' read -r handle loc; do
      [ -z "$handle" ] && continue
      gh api "users/$handle" --silent >/dev/null 2>&1 ||
        err "$loc: \"$handle\" isn't a GitHub username. Copy the username from an existing entry or website/config/custom.js."
    done < <(printf '%s' "$new_handles" | sort -u -t$'\t' -k1,1)
  else
    echo "NOTE   gh isn't authenticated, so new GitHub usernames weren't verified."
  fi
fi

# Whitespace-only edits make the diff harder to review.
total=$(g diff --numstat "$base" -- handbook/ | awk '{ n += $1 + $2 } END { print n + 0 }')
ignoring_ws=$(g diff -w --numstat "$base" -- handbook/ | awk '{ n += $1 + $2 } END { print n + 0 }')
[ "$ignoring_ws" -lt "$total" ] &&
  warn "Some changes are whitespace-only ($((total - ignoring_ws)) diff lines). Revert whitespace changes the request didn't ask for. Compare git diff with git diff -w to find them."

echo
echo "$errors error(s), $warnings warning(s)."
[ "$errors" -eq 0 ]
