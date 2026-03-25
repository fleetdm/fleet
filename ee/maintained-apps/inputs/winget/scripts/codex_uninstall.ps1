# Removes the user-scope layout created by codex_install.ps1

$ErrorActionPreference = "Continue"
$installRoot = Join-Path $env:LOCALAPPDATA "Programs\Codex CLI"

if (Test-Path -LiteralPath $installRoot) {
    Remove-Item -LiteralPath $installRoot -Recurse -Force
}

Exit 0
