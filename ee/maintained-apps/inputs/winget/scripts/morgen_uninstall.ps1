# Attempts to locate this application's NSIS uninstaller from the registry and run it silently.

$displayName = "Morgen"
$publisher = "Morgen AG"

$paths = @(
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    ($_.DisplayName -eq $displayName -or $_.DisplayName -like "$displayName *") -and
    ($publisher -eq "" -or $_.Publisher -like "*$publisher*")
  }
  if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall -or -not $uninstall.UninstallString) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

try {
    $uninstallString = $uninstall.UninstallString
    $exePath = ""
    if ($uninstallString -match '^"([^"]+)"') {
        $exePath = $matches[1]
    } elseif ($uninstallString -match '^(.+?\.exe)') {
        $exePath = $matches[1]
    } else {
        $exePath = $uninstallString
    }

    $installDir = $uninstall.InstallLocation
    if (-not $installDir -or -not (Test-Path $installDir)) {
        $installDir = Split-Path -Parent $exePath
    }

    $argumentList = @("/S", "_?=$installDir")

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

    # Only sweep leftovers on a successful uninstall, and never a root/short path
    if ($exitCode -eq 0 -and $installDir) {
        $resolvedDir = $null
        try { $resolvedDir = (Resolve-Path -LiteralPath $installDir -ErrorAction Stop).Path } catch { $resolvedDir = $null }
        if ($resolvedDir -and ($resolvedDir -match '^[A-Za-z]:\\') -and ((($resolvedDir.TrimEnd('\')) -split '\\').Count -ge 3) -and (Test-Path -LiteralPath $resolvedDir)) {
            Remove-Item -LiteralPath $resolvedDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }

    Exit $exitCode
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
