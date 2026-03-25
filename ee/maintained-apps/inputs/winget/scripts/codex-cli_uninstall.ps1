# Removes layouts created by codex-cli_install.ps1 (machine and per-user fallbacks).

$ErrorActionPreference = "Continue"
foreach ($installRoot in @(
    (Join-Path $env:ProgramFiles "Codex CLI"),
    (Join-Path $env:LOCALAPPDATA "Programs\Codex CLI")
)) {
    if (Test-Path -LiteralPath $installRoot) {
        Remove-Item -LiteralPath $installRoot -Recurse -Force
    }
}

Exit 0
