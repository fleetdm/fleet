# Uninstall Mozilla Thunderbird (x64 en-US) via registry; NSIS silent /S

$paths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null
foreach ($p in $paths) {
    $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
        $_.DisplayName -and
        $_.DisplayName -like '*Thunderbird*' -and
        $_.DisplayName -like '*(x64*' -and
        $_.Publisher -and
        $_.Publisher -like '*Mozilla*'
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

Stop-Process -Name "thunderbird" -Force -ErrorAction SilentlyContinue

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
