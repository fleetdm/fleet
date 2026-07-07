# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# AnyDesk uses its own installer; NSIS-style /S does nothing. The silent
# machine-wide install (per the winget manifest) is:
#   AnyDesk.exe --install <path> --silent
# The x86 build is the only one published and installs under Program Files (x86).

$exeFilePath = "${env:INSTALLER_PATH}"

try {

$installPath = "${env:ProgramFiles(x86)}\AnyDesk"

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "--install `"$installPath`" --silent --create-shortcuts"
  PassThru = $true
  Wait = $true
}

# Start process and track exit code
$process = Start-Process @processOptions
$exitCode = $process.ExitCode

# Prints the exit code
Write-Host "Install exit code: $exitCode"
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
