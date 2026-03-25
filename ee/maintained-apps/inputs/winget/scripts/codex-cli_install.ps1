# Codex ships as a .zip of portable binaries (winget NestedInstallerType: portable).
# INSTALLER_PATH points at the downloaded .zip.
# Prefer Program Files (matches Fleet validation + managed installs); fall back to %LOCALAPPDATA% without admin.

$ErrorActionPreference = "Stop"
$zipPath = "${env:INSTALLER_PATH}"
$machineRoot = Join-Path $env:ProgramFiles "Codex CLI"
$userRoot = Join-Path $env:LOCALAPPDATA "Programs\Codex CLI"
$extractDir = Join-Path $env:TEMP ("codex-winget-extract-" + [Guid]::NewGuid().ToString())

if (-not (Test-Path -LiteralPath $machineRoot)) {
    try {
        New-Item -ItemType Directory -Path $machineRoot -Force -ErrorAction Stop | Out-Null
        $installRoot = $machineRoot
    } catch {
        New-Item -ItemType Directory -Path $userRoot -Force | Out-Null
        $installRoot = $userRoot
    }
} else {
    $installRoot = $machineRoot
}

try {
    if (-not (Test-Path -LiteralPath $zipPath)) {
        Write-Host "Installer not found: $zipPath"
        Exit 1
    }

    New-Item -ItemType Directory -Path $installRoot -Force | Out-Null
    New-Item -ItemType Directory -Path $extractDir -Force | Out-Null
    Expand-Archive -LiteralPath $zipPath -DestinationPath $extractDir -Force

    # Matches winget manifest NestedInstallerFiles for x64
    $mainExe = Join-Path $extractDir "codex-x86_64-pc-windows-msvc.exe"
    if (-not (Test-Path -LiteralPath $mainExe)) {
        Write-Host "Expected binary codex-x86_64-pc-windows-msvc.exe not found in archive"
        Exit 1
    }

    $destExe = Join-Path $installRoot "codex.exe"
    Copy-Item -LiteralPath $mainExe -Destination $destExe -Force

    foreach ($extra in @(
        "codex-command-runner.exe",
        "codex-windows-sandbox-setup.exe"
    )) {
        $src = Join-Path $extractDir $extra
        if (Test-Path -LiteralPath $src) {
            Copy-Item -LiteralPath $src -Destination (Join-Path $installRoot $extra) -Force
        }
    }

    Exit 0
} catch {
    Write-Host "Error: $_"
    Exit 1
} finally {
    Remove-Item -LiteralPath $extractDir -Recurse -Force -ErrorAction SilentlyContinue
}
