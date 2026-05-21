#!/bin/sh
# PostToolUse hook: run prettier on frontend files after Edit/Write
# Receives tool event JSON on stdin

INPUT=$(cat)
# Extract file_path with grep to avoid jq parse errors from control chars in tool input
FILE_PATH=$(printf '%s' "$INPUT" | grep -o '"file_path"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/.*"file_path"[[:space:]]*:[[:space:]]*"//;s/"$//')

if [ -z "$FILE_PATH" ]; then
  exit 0
fi

case "$FILE_PATH" in
  *.ts|*.tsx|*.scss|*.css|*.js|*.jsx)
    # Use local prettier (avoid npx auto-install over network)
    if [ -x ./node_modules/.bin/prettier ]; then
      ./node_modules/.bin/prettier --write "$FILE_PATH" 2>/dev/null
    elif command -v npx >/dev/null 2>&1 && npx --no-install prettier --version >/dev/null 2>&1; then
      npx --no-install prettier --write "$FILE_PATH" 2>/dev/null
    fi
    ;;
esac

exit 0
