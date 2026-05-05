# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Ollama uses an Inno Setup-based installer with user-scope installation.
# The installer spawns a persistent "ollama serve" background process after
# install, which prevents Start-Process -Wait from ever returning. We start
# without -Wait and poll the installer process with a timeout instead.
#
# /SP- suppresses "setup is already running" prompt.
# /CURRENTUSER ensures user-scope install without elevation prompt.
# /CLOSEAPPLICATIONS closes running instances before install.
# /MERGETASKS=!runcode prevents auto-launching the app after install.
$process = Start-Process -FilePath "$exeFilePath" `
  -ArgumentList "/SP- /VERYSILENT /SUPPRESSMSGBOXES /NORESTART /CURRENTUSER /CLOSEAPPLICATIONS /MERGETASKS=!runcode" `
  -PassThru

# Wait up to 3 minutes for the installer process itself to exit.
$timeoutSeconds = 180
$exited = $process.WaitForExit($timeoutSeconds * 1000)

if (-not $exited) {
  Write-Host "Installer process did not exit within ${timeoutSeconds}s, stopping it."
  Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
  # Give it a moment then check if Ollama was actually installed
  Start-Sleep -Seconds 2
}

$exitCode = $process.ExitCode
Write-Host "Install exit code: $exitCode"

# Stop any Ollama background processes spawned by the installer so the
# script can return cleanly.
Stop-Process -Name "ollama app" -Force -ErrorAction SilentlyContinue
Stop-Process -Name "ollama" -Force -ErrorAction SilentlyContinue

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
