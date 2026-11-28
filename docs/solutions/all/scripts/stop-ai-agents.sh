#!/bin/bash
# Stops common AI coding agents on macOS and Linux and keeps them from starting
# again until restore-ai-agents.sh runs. Each agent's command is renamed with a
# .fleet-disabled suffix, so the restore script can put it back.
#
# Runs as root through Fleet. A user who owns an install (for example, in
# ~/.local/bin) can rename it back, so pair this with conditional access.

SUFFIX=".fleet-disabled"
COMMANDS="claude codex copilot cursor-agent gemini"

if [ "$(uname)" = "Darwin" ]; then
  HOMES=$(dscl . -list /Users NFSHomeDirectory | awk '$2 ~ /^\/Users\// {print $2}')
else
  HOMES=$(awk -F: '$3 >= 1000 && $6 ~ /^\/home\// {print $6}' /etc/passwd)
fi

candidates() {
  for cmd in $COMMANDS; do
    echo "/usr/local/bin/$cmd"
    echo "/opt/homebrew/bin/$cmd"
    for home in $HOMES; do
      echo "$home/.local/bin/$cmd"
      echo "$home/.npm-global/bin/$cmd"
      echo "$home/.bun/bin/$cmd"
      for nvm_bin in "$home"/.nvm/versions/node/*/bin; do
        echo "$nvm_bin/$cmd"
      done
    done
  done
  if [ "$(uname)" = "Darwin" ]; then
    echo "/Applications/Cursor.app"
    for home in $HOMES; do
      echo "$home/Applications/Cursor.app"
    done
  fi
}

disabled=0
while IFS= read -r path; do
  # -L catches symlinks whose target is already gone.
  if [ -e "$path" ] || [ -L "$path" ]; then
    if mv "$path" "$path$SUFFIX"; then
      echo "Disabled: $path"
      disabled=$((disabled + 1))
    else
      echo "Failed to disable: $path"
    fi
  fi
done < <(candidates)

# npm installs run as "node .../<package>/...", so match on the package path too.
for cmd in $COMMANDS; do
  pkill -x "$cmd"
done
pkill -x Cursor
pkill -f '@anthropic-ai/claude-code'
pkill -f '@openai/codex'
pkill -f '@github/copilot'
pkill -f '@google/gemini-cli'
pkill -f 'cursor-agent'

echo "Disabled $disabled AI agent install(s) and stopped running agents."
exit 0
