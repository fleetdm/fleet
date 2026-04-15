# Install Claude Code using the official installer
# https://code.claude.com/docs/en/quickstart
# Fleet runs scripts as SYSTEM.

try {
  irm https://claude.ai/install.ps1 | iex

  Write-Host "Claude Code installed successfully."
  Exit 0
} catch {
  Write-Host "Error: $_"
  Exit 1
}
