# Attempts to locate Focusrite Control 2's Inno Setup uninstaller and run it silently.
# Focusrite Control 2 is machine-scope.

$displayName = "Focusrite Control 2"
$publisher   = "Focusrite"

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

$installDir = if ($uninstall.InstallLocation -and (Test-Path -LiteralPath $uninstall.InstallLocation)) {
    $uninstall.InstallLocation.TrimEnd('\')
} else {
    (Split-Path -Parent $exePath).TrimEnd('\')
}

# Inno Setup silent uninstall switches
$argumentList = @("/VERYSILENT", "/NORESTART")

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
