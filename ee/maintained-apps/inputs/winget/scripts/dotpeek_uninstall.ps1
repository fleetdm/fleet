# Removes JetBrains dotPeek via its registry uninstall entry. The JetBrains
# uninstaller takes /Silent=True rather than the NSIS /S.
# https://resharper-support.jetbrains.com/hc/en-us/articles/207241485

$softwareNameLike = "JetBrains dotPeek*"
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

[array]$selected = $uninstallKeys | Where-Object {
    $_.DisplayName -and
    $_.DisplayName -like $softwareNameLike -and
    $_.Publisher -like $publisherLike -and
    $_.UninstallString
}

if ($selected.Count -eq 0) {
    Write-Host "Uninstall entry not found for $softwareNameLike; nothing to do."
    Exit 0
}

Stop-Process -Name "dotPeek32" -Force -ErrorAction SilentlyContinue
Stop-Process -Name "dotPeek64" -Force -ErrorAction SilentlyContinue
Stop-Process -Name "dotPeek64a" -Force -ErrorAction SilentlyContinue

foreach ($entry in $selected) {
    $uninstallCommand = $entry.UninstallString

    # JetBrains stores unquoted paths that contain spaces.
    $exePath = ""
    $existingArgs = ""
    if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
        $exePath = $matches[1]
        $existingArgs = $matches[2].Trim()
    } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
        $exePath = $matches[1]
        $existingArgs = $matches[2].Trim()
    } elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
        $exePath = $matches[1]
        $existingArgs = $matches[2].Trim()
    } else {
        Write-Host "Could not parse uninstall string: $uninstallCommand"
        $exitCode = 1
        continue
    }

    if ($existingArgs -notmatch '(?i)/Silent=') {
        $existingArgs = ("$existingArgs /Silent=True").Trim()
    }

    Write-Host "Selected entry DisplayName: $($entry.DisplayName)"
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
    Write-Host "Uninstall exit code: $($process.ExitCode)"
    if ($process.ExitCode -ne 0) {
        $exitCode = $process.ExitCode
    }
}

Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
