# Local build script: publishes Fleet Desktop and packages it into an MSI.
# Run from a Developer PowerShell or a regular PowerShell on Windows.
#
# Usage:
#   .\build.ps1                       # builds Release MSI
#   .\build.ps1 -Configuration Debug  # builds Debug MSI
#   .\build.ps1 -SkipMsi              # publishes EXE only, skips MSI
#   .\build.ps1 -MsiOnly              # builds MSI from the existing publish dir
#
# Signing happens in CI (.github/workflows/fleet-desktop-windows-build.yml):
# -SkipMsi, sign the EXE, -MsiOnly, sign the MSI. -MsiOnly doesn't republish,
# so the signed EXE placed in the publish dir is what gets packaged.

[CmdletBinding()]
param(
    [ValidateSet("Release", "Debug")]
    [string]$Configuration = "Release",
    [switch]$SkipMsi,
    [switch]$MsiOnly
)

$ErrorActionPreference = "Stop"
if ($SkipMsi -and $MsiOnly) { throw "-SkipMsi and -MsiOnly are mutually exclusive" }
$RepoRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$AppProj  = Join-Path $RepoRoot "FleetDesktop\FleetDesktop.csproj"
$WixProj  = Join-Path $RepoRoot "Installer\FleetDesktop.Installer.wixproj"

# Resolve version from the .csproj <Version> element so the MSI and EXE stay in lockstep.
$xml = [xml](Get-Content $AppProj)
$Version = ($xml.Project.PropertyGroup | Where-Object { $_.Version } | Select-Object -First 1).Version
if (-not $Version) { throw "Could not read <Version> from $AppProj" }
Write-Host "==> Fleet Desktop v$Version ($Configuration)" -ForegroundColor Cyan

$PublishDir = Join-Path $RepoRoot "FleetDesktop\bin\$Configuration\net8.0-windows\win-x64\publish\"
$ExePath = Join-Path $PublishDir "FleetDesktop.exe"

# 1) Publish the WPF app as a self-contained single-file EXE.
if (-not $MsiOnly) {
    Write-Host "==> Publishing FleetDesktop.exe..." -ForegroundColor Cyan
    dotnet publish $AppProj `
        -c $Configuration `
        -r win-x64 `
        --self-contained true `
        -p:PublishSingleFile=true `
        -p:IncludeNativeLibrariesForSelfExtract=true `
        -p:EnableCompressionInSingleFile=true `
        -p:Version=$Version
    if ($LASTEXITCODE -ne 0) { throw "dotnet publish failed" }
}

if (-not (Test-Path $ExePath)) { throw "Expected $ExePath to exist" }
Write-Host "    EXE: $ExePath" -ForegroundColor Green

if ($env:GITHUB_OUTPUT) {
    Add-Content -Path $env:GITHUB_OUTPUT -Value "EXE_PATH=$ExePath"
    Add-Content -Path $env:GITHUB_OUTPUT -Value "PUBLISH_DIR=$PublishDir"
    Add-Content -Path $env:GITHUB_OUTPUT -Value "VERSION=$Version"
}

if ($SkipMsi) {
    Write-Host "==> Skipping MSI build (-SkipMsi). Done." -ForegroundColor Green
    exit 0
}

# 2) Build the MSI.
Write-Host "==> Building MSI..." -ForegroundColor Cyan
dotnet build $WixProj `
    -c $Configuration `
    -p:Version=$Version `
    -p:PublishDir=$PublishDir
if ($LASTEXITCODE -ne 0) { throw "MSI build failed" }

$MsiPath = Join-Path $RepoRoot "Installer\bin\$Configuration\fleet_desktop-v$Version.msi"
if (-not (Test-Path $MsiPath)) {
    # WiX may emit under x64 subdir depending on platform setup.
    $MsiPath = Join-Path $RepoRoot "Installer\bin\x64\$Configuration\fleet_desktop-v$Version.msi"
}
if (-not (Test-Path $MsiPath)) { throw "MSI build succeeded but expected output file was not found" }
Write-Host "    MSI: $MsiPath" -ForegroundColor Green

Write-Host ""
Write-Host "==> Done." -ForegroundColor Green
Write-Host "    $MsiPath"

# Output for GitHub Actions
if ($env:GITHUB_OUTPUT) {
    Add-Content -Path $env:GITHUB_OUTPUT -Value "MSI_PATH=$MsiPath"
    Add-Content -Path $env:GITHUB_OUTPUT -Value "MSI_NAME=$(Split-Path -Leaf $MsiPath)"
}
