$ExpectedExitCodes = @(0, 1641, 3010, 1223)

# Power Automate for desktop is a WiX Burn bundle installed by the
# Setup.Microsoft.PowerAutomate.exe bootstrapper. The visible ARP entry is the inner
# MSI, whose UninstallString is "MsiExec.exe /I{GUID}" -- that is install/repair, not
# an uninstall, and the bootstrapper's -Uninstall/-Silent switches are not valid for
# MsiExec.exe. The documented silent uninstall runs the bootstrapper itself:
#   Setup.Microsoft.PowerAutomate.exe -Silent -Uninstall
# https://learn.microsoft.com/power-automate/desktop-flows/install-silently

# Stop running PAD processes so the uninstall isn't blocked.
Stop-Process -Name "PAD.Console.Host" -Force -ErrorAction SilentlyContinue

$setupExe = $null

# 1) The Burn bundle's ARP entry records the cached bootstrapper in BundleCachePath.
$arpPaths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)
foreach ($p in $arpPaths) {
  $bundle = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -like 'Power Automate for desktop*' -and $_.BundleCachePath
  } | Select-Object -First 1
  if ($bundle) { $setupExe = $bundle.BundleCachePath; break }
}

# 2) Fall back to searching the Burn package cache for the bootstrapper.
if (-not $setupExe -or -not (Test-Path $setupExe)) {
  $cache = Join-Path $env:ProgramData 'Package Cache'
  $found = Get-ChildItem -Path $cache -Recurse -Filter 'Setup.Microsoft.PowerAutomate.exe' -ErrorAction SilentlyContinue | Select-Object -First 1
  if ($found) { $setupExe = $found.FullName }
}

if (-not $setupExe -or -not (Test-Path $setupExe)) {
  Write-Host "Power Automate bootstrapper not found; nothing to uninstall."
  Exit 0
}

Write-Host "Uninstall command: $setupExe"
Write-Host "Uninstall args: -Silent -Uninstall"

try {
    $processOptions = @{
        FilePath = $setupExe
        ArgumentList = @("-Silent", "-Uninstall")
        NoNewWindow = $true
        PassThru = $true
        Wait = $true
    }
    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Uninstall exit code: $exitCode"
    if ($ExpectedExitCodes -contains $exitCode) { Exit 0 }
    Exit $exitCode
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
