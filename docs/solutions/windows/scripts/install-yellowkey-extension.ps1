<#
.SYNOPSIS
    Installs and loads the windows_yellowkey osquery extension on this host.

.DESCRIPTION
    Fleet run_script remediation for the windows-yellowkey-extension
    policy. Wrapper that fetches the canonical installer from
    allenhouchins/fleet-extensions and executes it. The full install
    logic (download, PE-header check, service stop, kill lingering
    child, hardened ACLs, loader write, service restart) lives in
    that upstream script; this wrapper exists only because Fleet's
    GitOps run_script needs a file on disk to upload.

    Update workflow: none. Allen's CI republishes the binary on every
    push to main, and the upstream installer always pulls from
    releases/latest/download, so this file never needs editing.

.OUTPUTS
    Whatever the upstream installer writes to stdout.

.NOTES
    Exit codes are pass-through from the upstream installer:
      0 = Installed; service back to Running
      3 = Fleet osquery service not present
      4 = Filesystem operation failed
      5 = Service did not return to Running
      6 = Download failed or asset is not a valid PE32+ executable
      8 = Unsupported architecture
    Additional codes from the wrapper itself:
      90 = Could not fetch the upstream installer
#>

[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'

$UpstreamUrl = 'https://raw.githubusercontent.com/allenhouchins/fleet-extensions/main/windows_yellowkey/install-windows-yellowkey-extension.ps1'
$LocalPath   = Join-Path $env:TEMP "install-windows-yellowkey-$([guid]::NewGuid()).ps1"

Write-Output "=== windows_yellowkey installer (wrapper) ==="
Write-Output "Upstream: $UpstreamUrl"
Write-Output ""

try {
    try {
        Invoke-WebRequest -Uri $UpstreamUrl -OutFile $LocalPath -UseBasicParsing -TimeoutSec 60
    } catch {
        Write-Output "FAIL: could not fetch the upstream installer: $($_.Exception.Message)"
        exit 90
    }

    & powershell.exe -ExecutionPolicy Bypass -NoProfile -File $LocalPath
    exit $LASTEXITCODE
} finally {
    Remove-Item -Path $LocalPath -Force -ErrorAction SilentlyContinue
}
