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
$registered = $false

# The Burn bundle's machine-wide (HKLM) ARP entry records the cached bootstrapper
# in BundleCachePath.
$arpPaths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)
foreach ($p in $arpPaths) {
  $entries = @(Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -like 'Power Automate for desktop*'
  })
  if ($entries.Count -gt 0) { $registered = $true }
  $bundle = $entries | Where-Object {
    $_.BundleCachePath -and (Test-Path -LiteralPath $_.BundleCachePath)
  } | Select-Object -First 1
  if ($bundle) { $setupExe = $bundle.BundleCachePath; break }
}

if (-not $setupExe) {
  if ($registered) {
    Write-Host "Power Automate is registered, but its bootstrapper was not found."
    Exit 1
  }
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
