# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Evernote uses an electron-builder NSIS installer. It defaults to a per-user
# install, so /allusers is required for a machine-wide install alongside the
# NSIS /S silent flag.
#
# The NSIS self-extractor intermittently dies with 0xc0000005 (access
# violation, exit code -1073741819) before extracting anything. The same
# installer succeeds on the next run, so retry on that specific exit code
# only; every other exit code is reported as-is.
$maxAttempts = 3
$exitCode = 1

for ($attempt = 1; $attempt -le $maxAttempts; $attempt++) {
  $processOptions = @{
    FilePath = "$exeFilePath"
    ArgumentList = "/allusers /S"
    PassThru = $true
    Wait = $true
  }

  $process = Start-Process @processOptions
  $exitCode = $process.ExitCode

  Write-Host "Install exit code: $exitCode (attempt $attempt of $maxAttempts)"

  if ($exitCode -ne -1073741819) {
    Exit $exitCode
  }

  Start-Sleep -Seconds 10
}

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
