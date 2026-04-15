# Install Claude Code via winget
# Fleet runs scripts as SYSTEM; winget must be resolved manually.

# Resolve the winget executable path (not on PATH when running as SYSTEM)
$ResolveWingetPath = Resolve-Path "C:\Program Files\WindowsApps\Microsoft.DesktopAppInstaller_*_x64__8wekyb3d8bbwe" -ErrorAction SilentlyContinue
if ($ResolveWingetPath) {
  $WingetPath = $ResolveWingetPath[-1].Path
} else {
  Write-Host "Error: winget (App Installer) is not available on this system."
  Exit 1
}

$WingetExe = Join-Path $WingetPath "winget.exe"
if (-not (Test-Path $WingetExe)) {
  Write-Host "Error: winget.exe not found at $WingetExe"
  Exit 1
}

try {
  # Check if Claude Code is already installed
  $installed = & $WingetExe list --id Anthropic.ClaudeCode --exact --accept-source-agreements 2>&1
  if ($LASTEXITCODE -eq 0 -and ($installed | Select-String "Anthropic.ClaudeCode")) {
    Write-Host "Claude Code is already installed. Upgrading..."
    & $WingetExe upgrade --id Anthropic.ClaudeCode --exact --silent --accept-package-agreements --accept-source-agreements --disable-interactivity
  } else {
    & $WingetExe install --id Anthropic.ClaudeCode --exact --silent --accept-package-agreements --accept-source-agreements --disable-interactivity
  }

  $exitCode = $LASTEXITCODE
  if ($exitCode -ne 0) {
    Write-Host "Error: winget operation failed with exit code $exitCode"
    Exit $exitCode
  }

  Write-Host "Claude Code installed successfully."
  Exit 0
} catch {
  Write-Host "Error: $_"
  Exit 1
}
