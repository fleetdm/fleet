#!/bin/sh
# PostToolUse hook: auto-fix lint issues, then report anything remaining
# Runs after formatters (goimports, prettier) so it only sees convention violations

INPUT=$(cat)
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')

if [ -z "$FILE_PATH" ]; then
  exit 0
fi

case "$FILE_PATH" in
  *.go)
    # Skip third_party
    case "$FILE_PATH" in
      */third_party/*) exit 0 ;;
    esac

    PKG_DIR=$(dirname "$FILE_PATH")

    if command -v golangci-lint >/dev/null 2>&1; then
      # First pass: auto-fix what we can
      golangci-lint run --fix "$PKG_DIR/..." 2>/dev/null

      # Second pass: report anything that couldn't be auto-fixed
      RESULT=$(golangci-lint run "$PKG_DIR/..." 2>&1 | grep -v "^level=" | head -20)
      if [ -n "$RESULT" ]; then
        printf '{"additionalContext": "golangci-lint found issues that need manual fixes in %s:\\n%s"}' "$FILE_PATH" "$RESULT"
      fi
    fi
    ;;

  *.ts|*.tsx)
    if command -v npx >/dev/null 2>&1; then
      # First pass: auto-fix
      npx eslint --fix "$FILE_PATH" 2>/dev/null

      # Second pass: report remaining issues
      RESULT=$(npx eslint "$FILE_PATH" 2>/dev/null | head -20)
      if [ -n "$RESULT" ] && echo "$RESULT" | grep -q "error\|warning"; then
        printf '{"additionalContext": "ESLint found issues that need manual fixes in %s:\\n%s"}' "$FILE_PATH" "$RESULT"
      fi
    fi
    ;;
esac

exit 0
