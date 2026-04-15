#!/bin/bash
set -e

# Install Claude Code via npm
# Requires Node.js and npm to be installed on the host

if ! command -v npm &> /dev/null; then
  echo "Error: npm is not installed. Please install Node.js first."
  exit 1
fi

npm install -g @anthropic-ai/claude-code
