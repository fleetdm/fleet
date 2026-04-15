# Install Claude Code via npm
# Requires Node.js and npm to be installed on the host

try {
  $npmPath = Get-Command npm -ErrorAction Stop | Select-Object -ExpandProperty Source
} catch {
  Write-Host "Error: npm is not installed. Please install Node.js first."
  Exit 1
}

try {
  npm install -g @anthropic-ai/claude-code
  $exitCode = $LASTEXITCODE
  if ($exitCode -ne 0) {
    Write-Host "Error: npm install failed with exit code $exitCode"
    Exit $exitCode
  }
  Write-Host "Claude Code installed successfully."
  Exit 0
} catch {
  Write-Host "Error: $_"
  Exit 1
}
