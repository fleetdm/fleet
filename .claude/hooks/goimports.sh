#!/bin/sh
# PostToolUse hook: run goimports on Go files after Edit/Write
# Receives tool event JSON on stdin

INPUT=$(cat)
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')

if [ -z "$FILE_PATH" ]; then
  exit 0
fi

case "$FILE_PATH" in
  *.go)
    if command -v goimports >/dev/null 2>&1; then
      goimports -w "$FILE_PATH" 2>/dev/null
    elif command -v gofumpt >/dev/null 2>&1; then
      gofumpt -w "$FILE_PATH" 2>/dev/null
    else
      gofmt -w "$FILE_PATH" 2>/dev/null
    fi
    ;;
esac

exit 0
