# 1Password Uninstall Script
# Uses winget to uninstall the package silently and non-interactively

# Check if winget is available
if (-not (Get-Command winget -ErrorAction SilentlyContinue)) {
    Write-Host "Error: winget is not available on this system"
    Exit 1
}

# Uninstall using winget with silent and non-interactive flags
winget uninstall --id AgileBits.1Password `
    --silent `
    --disable-interactivity

# Verify the uninstall was successful by checking if the package is still installed
# winget list returns exit code 0 if package is found, non-zero if not found
$null = winget list --id AgileBits.1Password --exact 2>&1
if ($LASTEXITCODE -eq 0) {
    # Package is still installed, uninstall failed
    Write-Host "Error: Package is still installed after uninstall attempt"
    Exit 1
} else {
    # Package not found, uninstall succeeded
    Write-Host "Successfully uninstalled AgileBits.1Password"
    Exit 0
}

