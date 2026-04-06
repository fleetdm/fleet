#!/bin/sh
# PostToolUse hook: auto-fix lint issues, then report anything remaining
# Runs golangci-lint on the affected package (not make lint-go-incremental, which is too
# slow for a PostToolUse hook). Runs after formatters (goimports, prettier) so it only
# sees convention violations.

INPUT=$(cat)
# Extract file_path with grep to avoid jq parse errors from control chars in tool input
FILE_PATH=$(printf '%s' "$INPUT" | grep -o '"file_path"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/.*"file_path"[[:space:]]*:[[:space:]]*"//;s/"$//')

if [ -z "$FILE_PATH" ]; then
  exit 0
fi

# Need to be in the project root for make targets
PROJECT_DIR=$(printf '%s' "$INPUT" | grep -o '"cwd"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/.*"cwd"[[:space:]]*:[[:space:]]*"//;s/"$//')
if [ -z "$PROJECT_DIR" ]; then
  PROJECT_DIR="$CLAUDE_PROJECT_DIR"
fi
if [ -n "$PROJECT_DIR" ]; then
  cd "$PROJECT_DIR" || exit 0
fi

TMPFILE=$(mktemp)
trap 'rm -f "$TMPFILE"' EXIT

case "$FILE_PATH" in
  *.go)
    # Skip third_party (with or without leading path)
    case "$FILE_PATH" in
      third_party/*|*/third_party/*) exit 0 ;;
    esac

    # First pass: auto-fix what we can (uses golangci-lint directly for --fix)
    PKG_DIR=$(dirname "$FILE_PATH")
    if command -v golangci-lint >/dev/null 2>&1; then
      golangci-lint run --fix "$PKG_DIR/..." > /dev/null 2>&1
    fi

    # Second pass: lint the affected package (fast) and report remaining issues
    if command -v golangci-lint >/dev/null 2>&1; then
      golangci-lint run "$PKG_DIR/..." > "$TMPFILE" 2>&1
    else
      exit 0
    fi

    # Filter to real violations: path/to/file.go:LINE:COL: message (lintername)
    VIOLATIONS=$(grep -E '\.go:[0-9]+:[0-9]+:' "$TMPFILE" | head -20)

    if [ -n "$VIOLATIONS" ]; then
      echo "$VIOLATIONS" | jq -Rsc --arg fp "$FILE_PATH" \
        '{hookSpecificOutput: {hookEventName: "PostToolUse", additionalContext: ("golangci-lint found issues after editing " + $fp + ":\n" + .)}}'
    fi
    ;;

  *.ts|*.tsx)
    # Determine eslint binary (prefer local, avoid npx auto-install)
    if [ -x ./node_modules/.bin/eslint ]; then
      ESLINT="./node_modules/.bin/eslint"
    elif command -v npx >/dev/null 2>&1 && npx --no-install eslint --version >/dev/null 2>&1; then
      ESLINT="npx --no-install eslint"
    else
      exit 0
    fi

    if [ -n "$ESLINT" ]; then
      # First pass: auto-fix
      $ESLINT --fix "$FILE_PATH" > /dev/null 2>&1

      # Second pass: capture remaining issues (include stderr for config/parser errors)
      $ESLINT "$FILE_PATH" > "$TMPFILE" 2>&1

      if grep -q "error\|warning\|Error:" "$TMPFILE"; then
        jq -Rsc --arg fp "$FILE_PATH" \
          '{hookSpecificOutput: {hookEventName: "PostToolUse", additionalContext: ("ESLint found issues after editing " + $fp + ":\n" + .)}}' \
          < "$TMPFILE"
      fi
    fi
    ;;
esac

exit 0
