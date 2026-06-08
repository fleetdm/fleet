#!/bin/sh
# PostToolUse hook: auto-fix lint issues, then report anything remaining
# Uses the project's own make lint-go-incremental (only checks changes since branching from main)
# Runs after formatters (goimports, prettier) so it only sees convention violations

INPUT=$(cat)
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')

if [ -z "$FILE_PATH" ]; then
  exit 0
fi

# Need to be in the project root for make targets
PROJECT_DIR=$(echo "$INPUT" | jq -r '.cwd // empty')
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

    # Second pass: use project's incremental linter (only changes since branching from main)
    if [ -f Makefile ] && grep -q "lint-go-incremental" Makefile; then
      make lint-go-incremental > "$TMPFILE" 2>&1
    elif command -v golangci-lint >/dev/null 2>&1; then
      # Fallback if make target isn't available
      golangci-lint run "$PKG_DIR/..." > "$TMPFILE" 2>&1
    else
      exit 0
    fi

    # Filter out noise (level=warning, command echo, summary) and keep only real violations
    # Real violations look like: path/to/file.go:LINE:COL: message (lintername)
    VIOLATIONS=$(grep -v "^level=" "$TMPFILE" | grep -v "^\\./" | grep -v "^[0-9]* issues" | grep -v "^$" | grep -E '\.go:[0-9]+:[0-9]+:' | head -20)

    if [ -n "$VIOLATIONS" ]; then
      echo "$VIOLATIONS" | jq -Rsc --arg fp "$FILE_PATH" \
        '{hookSpecificOutput: {hookEventName: "PostToolUse", additionalContext: ("make lint-go-incremental found issues after editing " + $fp + ":\n" + .)}}'
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
