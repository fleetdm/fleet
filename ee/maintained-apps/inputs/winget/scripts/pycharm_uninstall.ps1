# Locates PyCharm Professional's NSIS uninstaller from the registry and runs it silently.
# JetBrains NSIS installers embed the version in DisplayName (e.g.
# "PyCharm 2024.3.5"). PyCharm Community Edition shares the "PyCharm " prefix
# (e.g. "PyCharm Community Edition 2025.2.6.1"), so we exclude it explicitly to
# avoid uninstalling the wrong edition.

$softwareNameLike = "PyCharm *"
$softwareNameExclude = "PyCharm Community Edition*"
$publisherLike = "*JetBrains*"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$exitCode = 0

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path $paths `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue }

$selected = $null
foreach ($key in $uninstallKeys) {
    if ($key.DisplayName `
        -and $key.DisplayName -like $softwareNameLike `
        -and $key.DisplayName -notlike $softwareNameExclude `
        -and $key.Publisher -like $publisherLike) {
        $selected = $key
        break
    }
}

if (-not $selected -or -not $selected.UninstallString) {
    Write-Host "Uninstall entry not found for $softwareNameLike (excluding $softwareNameExclude)"
    Exit 1
}

# Best-effort: stop the IDE so the uninstaller doesn't fail on locked files.
Stop-Process -Name "pycharm64" -Force -ErrorAction SilentlyContinue
Stop-Process -Name "pycharm" -Force -ErrorAction SilentlyContinue
Stop-Process -Name "fsnotifier" -Force -ErrorAction SilentlyContinue

$uninstallCommand = $selected.UninstallString

# Split the uninstall string into exe + args. Handle both quoted and unquoted
# exe paths.
$exePath = ""
$existingArgs = ""
if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
    # Quoted path: "C:\Path With Spaces\uninst.exe" [args]
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
    # Unquoted path that may contain spaces: capture through the .exe.
    # JetBrains stores e.g.
    # C:\Program Files\JetBrains\PyCharm 2024.3.5\bin\Uninstall.exe
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
    # Fallback: no .exe found, split on first whitespace.
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} else {
    Throw "Could not parse uninstall string: $uninstallCommand"
}

# NSIS uninstallers require /S for silent uninstall.
if ($existingArgs -notmatch '\b/S\b') {
    $existingArgs = ("$existingArgs /S").Trim()
}

Write-Host "Selected entry DisplayName: $($selected.DisplayName)"
Write-Host "Uninstall command: $exePath"
Write-Host "Uninstall args: $existingArgs"

$processOptions = @{
    FilePath = $exePath
    PassThru = $true
    Wait = $true
}

if ($existingArgs -ne '') {
    $processOptions.ArgumentList = $existingArgs
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode
Write-Host "Uninstall exit code: $exitCode"

Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
