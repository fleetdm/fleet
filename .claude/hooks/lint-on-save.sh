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
    # Skip third_party
    case "$FILE_PATH" in
      */third_party/*) exit 0 ;;
    esac

    # First pass: auto-fix what we can (uses golangci-lint directly for --fix)
    PKG_DIR=$(dirname "$FILE_PATH")
    if command -v golangci-lint >/dev/null 2>&1; then
      golangci-lint run --fix "$PKG_DIR/..." > /dev/null 2>&1
    fi

    # Second pass: use project's incremental linter (only changes since branching from main)
    LINT_FAILED=0
    if [ -f Makefile ] && grep -q "lint-go-incremental" Makefile; then
      make lint-go-incremental > "$TMPFILE" 2>&1
      LINT_EXIT=$?
    elif command -v golangci-lint >/dev/null 2>&1; then
      # Fallback if make target isn't available
      golangci-lint run "$PKG_DIR/..." > "$TMPFILE" 2>&1
      LINT_EXIT=$?
    else
      exit 0
    fi

    # Distinguish linter errors (broken setup) from lint violations (code issues)
    if grep -q "Error:" "$TMPFILE" && ! grep -q "\.go:" "$TMPFILE"; then
      # Linter itself errored (e.g., wrong Go version, missing binary) — report as a tool problem
      jq -Rsc \
        '{hookSpecificOutput: {hookEventName: "PostToolUse", additionalContext: ("Linter error (not a code issue — the linter setup may need fixing):\n" + .)}}' \
        < "$TMPFILE"
    elif grep -q "\.go:" "$TMPFILE"; then
      # Actual lint violations found in Go files
      jq -Rsc --arg fp "$FILE_PATH" \
        '{hookSpecificOutput: {hookEventName: "PostToolUse", additionalContext: ("make lint-go-incremental found issues after editing " + $fp + ":\n" + .)}}' \
        < "$TMPFILE"
    fi
    # If neither matched (e.g., "0 issues"), produce no output (clean)
    ;;

  *.ts|*.tsx)
    if command -v npx >/dev/null 2>&1; then
      # First pass: auto-fix
      npx eslint --fix "$FILE_PATH" 2>/dev/null

      # Second pass: capture remaining issues
      npx eslint "$FILE_PATH" > "$TMPFILE" 2>/dev/null

      if grep -q "error\|warning" "$TMPFILE"; then
        jq -Rsc --arg fp "$FILE_PATH" \
          '{hookSpecificOutput: {hookEventName: "PostToolUse", additionalContext: ("ESLint found issues after editing " + $fp + ":\n" + .)}}' \
          < "$TMPFILE"
      fi
    fi
    ;;
esac

exit 0
