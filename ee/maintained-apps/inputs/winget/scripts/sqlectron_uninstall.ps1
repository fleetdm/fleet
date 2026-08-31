# Locate the uninstall entry in the registry and run it silently.

$displayNameLike = "sqlectron*"
$publisher = "The Sqlectron Team"

$paths = @(
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -like $displayNameLike -and ($publisher -eq "" -or $_.Publisher -like "*$publisher*")
  }
  if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall -or -not $uninstall.UninstallString) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

try {
    $uninstallString = $uninstall.UninstallString

    # Parse the uninstaller executable path from the UninstallString.
    if ($uninstallString -match '^"([^"]+)"') {
        $uninstallExe = $matches[1]
    } elseif ($uninstallString -match '^(.+?\.exe)') {
        $uninstallExe = $matches[1]
    } else {
        $uninstallExe = $uninstallString
    }

    # Determine the install directory for cleanup.
    $installDir = $uninstall.InstallLocation
    if (-not $installDir -or -not (Test-Path $installDir)) {
        $installDir = Split-Path $uninstallExe -Parent
    }

    $uninstallArgs = @("/S", "_?=$installDir")

    Write-Host "Uninstall command: $uninstallExe"
    Write-Host "Uninstall args: $uninstallArgs"

    $processOptions = @{
        FilePath = $uninstallExe
        ArgumentList = $uninstallArgs
        PassThru = $true
        Wait = $true
        NoNewWindow = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Uninstall exit code: $exitCode"

    if ($installDir -and (Test-Path $installDir)) {
        Remove-Item $installDir -Recurse -Force -ErrorAction SilentlyContinue
    }

    Exit $exitCode
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
