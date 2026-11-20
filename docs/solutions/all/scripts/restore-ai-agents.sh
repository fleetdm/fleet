#!/bin/bash
# Restores AI coding agents on macOS and Linux that stop-ai-agents.sh disabled.

SUFFIX=".fleet-disabled"

if [ "$(uname)" = "Darwin" ]; then
  HOMES=$(dscl . -list /Users NFSHomeDirectory | awk '$2 ~ /^\/Users\// {print $2}')
else
  HOMES=$(awk -F: '$3 >= 1000 && $6 ~ /^\/home\// {print $6}' /etc/passwd)
fi

DIRS="/usr/local/bin /opt/homebrew/bin /Applications"
for home in $HOMES; do
  DIRS="$DIRS $home/.local/bin $home/.npm-global/bin $home/.bun/bin $home/Applications"
  for nvm_bin in "$home"/.nvm/versions/node/*/bin; do
    DIRS="$DIRS $nvm_bin"
  done
done

restored=0
for dir in $DIRS; do
  for path in "$dir"/*"$SUFFIX"; do
    [ -e "$path" ] || [ -L "$path" ] || continue
    original="${path%"$SUFFIX"}"
    if [ -e "$original" ] || [ -L "$original" ]; then
      echo "Skipped, $original already exists: $path"
      continue
    fi
    if mv "$path" "$original"; then
      echo "Restored: $original"
      restored=$((restored + 1))
    else
      echo "Failed to restore: $path"
    fi
  done
done

echo "Restored $restored AI agent install(s)."
exit 0
