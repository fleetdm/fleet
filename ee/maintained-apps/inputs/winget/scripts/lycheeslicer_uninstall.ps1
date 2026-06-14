# Attempts to locate Lychee Slicer's NSIS (Nullsoft) uninstaller and run it silently.
# The registry DisplayName may include the version (e.g. "Lychee Slicer 7.6.5"), so match a prefix.

$displayName = "Lychee Slicer"
$publisher   = "Mango3D"

$paths = @(
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -and ($_.DisplayName -eq $displayName -or $_.DisplayName -like "$displayName*") -and
    ($publisher -eq "" -or $_.Publisher -like "*$publisher*")
  }
  if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall -or -not $uninstall.UninstallString) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

$uninstallString = $uninstall.UninstallString
$exePath = ""
if ($uninstallString -match '^"([^"]+)"(.*)') {
    $exePath = $matches[1]
} elseif ($uninstallString -match '^(.+?\.exe)(.*)$') {
    $exePath = $matches[1]
} else {
    Write-Host "Error: Could not parse uninstall string: $uninstallString"
    Exit 1
}

# Prefer the registry InstallLocation; fall back to the uninstaller's parent
# directory. _?=<installdir> must match $INSTDIR for NSIS to run synchronously in-place.
$installDir = if ($uninstall.InstallLocation -and (Test-Path -LiteralPath $uninstall.InstallLocation)) {
    $uninstall.InstallLocation.TrimEnd('\')
} else {
    (Split-Path -Parent $exePath).TrimEnd('\')
}

$argumentList = @("/S", "_?=$installDir")

Write-Host "Uninstall executable: $exePath"
Write-Host "Uninstall arguments: $($argumentList -join ' ')"

try {
    $processOptions = @{
        FilePath     = $exePath
        ArgumentList = $argumentList
        NoNewWindow  = $true
        PassThru     = $true
        Wait         = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Uninstall exit code: $exitCode"

    if (Test-Path -LiteralPath $installDir) {
        Remove-Item -LiteralPath $installDir -Recurse -Force -ErrorAction SilentlyContinue
    }

    Exit $exitCode
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
