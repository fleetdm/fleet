#!/bin/sh
# PreToolUse hook: block dangerous bash commands
# Exit 0 = allow, Exit 2 = block

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // empty')

if [ -z "$COMMAND" ]; then
  exit 0
fi

# Block rm -rf with dangerous targets (/, ~, *, bare . but not ./path)
echo "$COMMAND" | grep -qE 'rm\s+-rf\s+/' && {
  echo "BLOCKED: rm -rf with absolute path" >&2
  exit 2
}
echo "$COMMAND" | grep -qE 'rm\s+-rf\s+~' && {
  echo "BLOCKED: rm -rf home directory" >&2
  exit 2
}
echo "$COMMAND" | grep -qE 'rm\s+-rf\s+\*' && {
  echo "BLOCKED: rm -rf wildcard" >&2
  exit 2
}
echo "$COMMAND" | grep -qE 'rm\s+-rf\s+\.$' && {
  echo "BLOCKED: rm -rf current directory" >&2
  exit 2
}

# Block force push to main/master
echo "$COMMAND" | grep -qiE 'git\s+push\s+.*(--force|-f)\s+.*(main|master)' && {
  echo "BLOCKED: force push to main/master" >&2
  exit 2
}

# Block hard reset to remote
echo "$COMMAND" | grep -qiE 'git\s+reset\s+--hard\s+origin/' && {
  echo "BLOCKED: hard reset to remote" >&2
  exit 2
}

# Block pipe-to-shell
echo "$COMMAND" | grep -qiE '(curl|wget)\s+.*\|\s*(ba)?sh' && {
  echo "BLOCKED: pipe to shell" >&2
  exit 2
}

exit 0
