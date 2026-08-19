$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Beekeeper Studio ships an electron-builder NSIS installer. /S runs it
# silently; /allusers forces a per-machine install so the app is reachable
# from Fleet's SYSTEM context (matches the winget machine-scope switch).
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/S", "/allusers"
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
