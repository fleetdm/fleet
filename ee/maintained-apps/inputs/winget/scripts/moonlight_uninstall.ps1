# Attempts to locate Moonlight's WiX burn bundle uninstaller from the registry and run it silently.

$displayName = "Moonlight Game Streaming Client"
$publisher = "Moonlight Game Streaming Project"

$paths = @(
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    ($_.DisplayName -eq $displayName -or $_.DisplayName -like "$displayName*") -and
    ($publisher -eq "" -or $_.Publisher -like "*$publisher*")
  }
  if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall -or (-not $uninstall.UninstallString -and -not $uninstall.QuietUninstallString)) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

try {
    $uninstallString = if ($uninstall.QuietUninstallString) {
        $uninstall.QuietUninstallString
    } else {
        $uninstall.UninstallString
    }

    $exePath = ""
    if ($uninstallString -match '^"([^"]+)"') {
        $exePath = $matches[1]
    } elseif ($uninstallString -match '^(.+?\.exe)') {
        $exePath = $matches[1]
    } else {
        $exePath = $uninstallString
    }

    $argumentList = @("/uninstall", "/quiet", "/norestart")

    Write-Host "Uninstall command: $exePath"
    Write-Host "Uninstall args: $argumentList"

    $processOptions = @{
        FilePath = $exePath
        ArgumentList = $argumentList
        NoNewWindow = $true
        PassThru = $true
        Wait = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Uninstall exit code: $exitCode"
    Exit $exitCode
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
