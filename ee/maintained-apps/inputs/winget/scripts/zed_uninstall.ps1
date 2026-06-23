# Locates Zed's Inno Setup uninstaller from the registry and runs it silently.
# Zed's Inno installer has both machine and per-user modes baked in, so we
# search HKLM (machine, /ALLUSERS) AND HKCU (per-user fallback) -- this works
# regardless of which scope the install actually hit. Silent uninstall flags
# are Inno's: /VERYSILENT /SUPPRESSMSGBOXES /NORESTART. Inno removes its
# registry entry early in the uninstall, so detection clears even though the
# uninstaller detaches to %TEMP% to delete its own files.

# Match the exact product name (with a version-suffix fallback) and the exact
# publisher so we never uninstall an unrelated SKU that merely starts with "Zed".
$softwareName = "Zed"
$publisher    = "Zed Industries"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall'
)

# 0 = success; 3010/1641 = success but reboot required.
$ExpectedExitCodes = @(0, 3010, 1641)
$exitCode = 0

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path $paths `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue }

$selected = $null
foreach ($key in $uninstallKeys) {
    if ($key.DisplayName -and ($key.DisplayName -eq $softwareName -or $key.DisplayName -like "$softwareName*") -and $key.Publisher -eq $publisher) {
        $selected = $key
        break
    }
}

if (-not $selected -or -not $selected.UninstallString) {
    Write-Host "Uninstall entry not found for $softwareName"
    Exit 0
}

# Stop Zed and any helpers running out of the install dir so the uninstaller
# doesn't fail on locked files. Zed is Rust/GPUI (not Electron), so the main
# process is just Zed.exe -- but sweep by exe path under InstallLocation too
# in case future versions ship helpers.
Stop-Process -Name "Zed" -Force -ErrorAction SilentlyContinue
if ($selected.InstallLocation -and (Test-Path -LiteralPath $selected.InstallLocation)) {
    $loc = $selected.InstallLocation.TrimEnd('\')
    Get-Process | Where-Object { $_.Path -and $_.Path -like "$loc\*" } |
        ForEach-Object { Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue }
}
Start-Sleep -Seconds 2

# Prefer QuietUninstallString -- Inno populates it with /VERYSILENT
# /SUPPRESSMSGBOXES already, so we just add /NORESTART if missing.
$uninstallCommand = if ($selected.QuietUninstallString) {
    $selected.QuietUninstallString
} else {
    $selected.UninstallString
}

# Parse the uninstaller exe path. Inno usually quotes it because the install
# path often contains "Program Files" (or "AppData\Local\Programs" for the
# per-user fallback).
$exePath = ""
$existingArgs = ""
if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} else {
    Throw "Could not parse uninstall string: $uninstallCommand"
}

# Build the final argument list. Start with whatever Inno baked in
# (QuietUninstallString usually has /VERYSILENT /SUPPRESSMSGBOXES) and ensure
# all three silent-uninstall flags are present.
$argumentList = @()
if ($existingArgs) { $argumentList += ($existingArgs -split '\s+') }
foreach ($s in @("/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART")) {
    if ($argumentList -notcontains $s) { $argumentList += $s }
}

Write-Host "Selected entry DisplayName: $($selected.DisplayName)"
Write-Host "Uninstall command: $exePath"
Write-Host "Uninstall args: $($argumentList -join ' ')"

$processOptions = @{
    FilePath     = $exePath
    ArgumentList = $argumentList
    PassThru     = $true
    Wait         = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode
Write-Host "Uninstall exit code: $exitCode"

if ($ExpectedExitCodes -contains $exitCode) { Exit 0 }
Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
