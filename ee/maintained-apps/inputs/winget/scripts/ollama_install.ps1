# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Ollama uses an Inno Setup-based installer with user-scope installation.
# /SP- suppresses "setup is already running" prompt.
# /CURRENTUSER ensures user-scope install without elevation prompt.
# /CLOSEAPPLICATIONS closes running instances before install.
# /MERGETASKS=!runcode prevents auto-launching the app after install.
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/SP- /VERYSILENT /SUPPRESSMSGBOXES /NORESTART /CURRENTUSER /CLOSEAPPLICATIONS /MERGETASKS=!runcode"
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
