# Attempts to locate Granola's uninstaller from registry and execute it silently

$displayName = "Granola"
$productCode = "cdc80bd8-3b8c-5d86-a628-c46cf9da018d"

$paths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null

foreach ($p in $paths) {
    $codeCandidates = @(
        "$p\$productCode",
        "$p\{$productCode}"
    )

    foreach ($candidate in $codeCandidates) {
        if (Test-Path $candidate) {
            $uninstall = Get-ItemProperty -Path $candidate -ErrorAction SilentlyContinue
            if ($uninstall) {
                break
            }
        }
    }

    if ($uninstall) {
        break
    }

    $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
        $_.DisplayName -and ($_.DisplayName -eq $displayName -or $_.DisplayName -like "$displayName*")
    }

    if ($items) {
        $uninstall = $items | Select-Object -First 1
        break
    }
}

if (-not $uninstall) {
    Write-Host "Uninstall entry not found"
    Exit 0
}

$uninstallString = if ($uninstall.QuietUninstallString) {
    $uninstall.QuietUninstallString
}
else {
    $uninstall.UninstallString
}

if (-not $uninstallString) {
    Write-Host "Uninstall command not found"
    Exit 0
}

Stop-Process -Name "Granola" -Force -ErrorAction SilentlyContinue

$exePath = ""
$arguments = ""

if ($uninstallString -match '^"([^"]+)"(.*)') {
    $exePath = $matches[1]
    $arguments = $matches[2].Trim()
}
elseif ($uninstallString -match '^([^\s]+)(.*)') {
    $exePath = $matches[1]
    $arguments = $matches[2].Trim()
}
else {
    Write-Host "Error: Could not parse uninstall string: $uninstallString"
    Exit 1
}

$argumentList = @()
if ($arguments -ne '') {
    $argumentList += $arguments -split '\s+'
}

# NSIS uninstallers require /S for silent mode.
if ($argumentList -notcontains "/S" -and $arguments -notmatch '\b/S\b') {
    $argumentList += "/S"
}

Write-Host "Uninstall executable: $exePath"
Write-Host "Uninstall arguments: $($argumentList -join ' ')"

try {
    $processOptions = @{
        FilePath    = $exePath
        NoNewWindow = $true
        PassThru    = $true
        Wait        = $true
    }

    if ($argumentList.Count -gt 0) {
        $processOptions.ArgumentList = $argumentList
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode

    Write-Host "Uninstall exit code: $exitCode"
    Exit $exitCode
}
catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
