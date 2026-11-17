# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# PyCharm Community Edition ships as a Nullsoft (NSIS) installer; /S runs it silently.

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# In silent mode the installer refuses a non-empty install directory with a
# message box. Fleet runs it as SYSTEM, where nobody can dismiss that box, so a
# reinstall of the same version would hang. Empty the directory first (it's
# "PyCharm Community Edition <ProductVersion>"), stopping PyCharm Community Edition if it's running from there.
# Unsaved changes are lost.
$version = (Get-Item $exeFilePath).VersionInfo.ProductVersion
$programFiles = if (${env:ProgramW6432}) { ${env:ProgramW6432} } else { ${env:ProgramFiles} }
$installDir = Join-Path $programFiles "JetBrains\PyCharm Community Edition $version"

if ($version -and (Test-Path -LiteralPath $installDir)) {
    $running = Get-Process -ErrorAction SilentlyContinue |
        Where-Object { $_.Path -like "$installDir\*" }
    foreach ($proc in $running) {
        Write-Host "Stopping $($proc.Name) (PID $($proc.Id)) before installing."
        Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
    }
    $running | Wait-Process -Timeout 30 -ErrorAction SilentlyContinue

    # A stopped PyCharm Community Edition's exe and DLLs can stay locked for a few seconds.
    Write-Host "Removing the existing $installDir before reinstalling."
    for ($attempt = 0; $attempt -lt 30; $attempt++) {
        Remove-Item -LiteralPath $installDir -Recurse -Force -ErrorAction SilentlyContinue
        $left = @(Get-ChildItem -LiteralPath $installDir -Recurse -File -Force -ErrorAction SilentlyContinue)
        if ($left.Count -eq 0) { break }
        Start-Sleep -Seconds 2
    }
    if ($left.Count -gt 0) {
        Write-Host "Error: $($left.Count) file(s) in $installDir couldn't be removed (first: $($left[0].FullName)). The installer would hang on a non-empty directory."
        Exit 1
    }
}

# On Windows on Arm the installer refuses this x64 build with a message box
# that hangs under SYSTEM. It has no switch to skip that check, but skips it
# when TEAMCITY_VERSION is set. The app runs under x64 emulation.
$nativeArch = (Get-ItemProperty 'HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\Environment').PROCESSOR_ARCHITECTURE
if ($nativeArch -eq 'ARM64') {
  $env:TEAMCITY_VERSION = "1"
}

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/S"
  PassThru = $true
  Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

Write-Host "Install exit code: $exitCode"
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
