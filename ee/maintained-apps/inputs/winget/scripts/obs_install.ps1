# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Setup exits with code 2 while OBS is running, because it can't replace OBS's
# open files. Fleet runs as SYSTEM, outside the user's session, so it can't ask
# OBS to close: stop it first, along with its helpers (obs-browser-page.exe,
# obs-ffmpeg-mux.exe). Unsaved changes and active recordings or streams are lost.
$running = Get-Process -ErrorAction SilentlyContinue |
    Where-Object { $_.Name -in @("obs64", "obs32") -or $_.Path -like "*\obs-studio\*" }
foreach ($proc in $running) {
    Write-Host "Stopping $($proc.Name) (PID $($proc.Id)) before installing."
    Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
}
$running | Wait-Process -Timeout 30 -ErrorAction SilentlyContinue

# Add argument to install silently
# NSIS installers require /S flag for silent installation
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/S"
  PassThru = $true
  Wait = $true
}
    
# Start process and track exit code
$process = Start-Process @processOptions
$exitCode = $process.ExitCode

# Prints the exit code
Write-Host "Install exit code: $exitCode"
if ($exitCode -eq 6) {
    # Setup exits 6 when another app has OBS files loaded: the OBS Virtual
    # Camera in a video-call app or browser, or OBS game capture in a game.
    # Those apps aren't stopped here.
    Write-Host "OBS Studio setup stopped because other apps are using OBS files. Close them, then retry."
}
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}

