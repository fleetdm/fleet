# Paint.NET ships as a zip containing its installer .exe. Fleet downloads the zip
# to INSTALLER_PATH; this script extracts it and runs the installer silently.

$zipFilePath = "${env:INSTALLER_PATH}"

$installTimeoutSeconds = 420
$registrationTimeoutSeconds = 120

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

function Get-PaintDotNetEntry {
    Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -eq "Paint.NET" } |
        Select-Object -First 1
}

try {

$extractPath = Join-Path $env:TEMP "PaintDotNetInstall"
if (Test-Path $extractPath) { Remove-Item -Path $extractPath -Recurse -Force }
Expand-Archive -Path $zipFilePath -DestinationPath $extractPath -Force

$installer = Get-ChildItem -Path $extractPath -Filter "*.exe" -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
if (-not $installer) {
  Write-Host "Error: installer .exe not found under $extractPath"
  Exit 1
}

# /auto is the vendor's silent switch, per the winget manifest InstallerSwitches.
# -Wait also waits on descendants, so wait on the installer process alone.
$process = Start-Process -FilePath $installer.FullName -ArgumentList "/auto" -PassThru
# Keeps .ExitCode readable after the process ends.
$null = $process.Handle

$killed = $false
if (-not $process.WaitForExit($installTimeoutSeconds * 1000)) {
  Write-Host "Installer process did not exit within ${installTimeoutSeconds}s, stopping it."
  Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
  $null = $process.WaitForExit(30 * 1000)
  $killed = $true
}

$exitCode = $null
if ($process.HasExited) {
  $exitCode = $process.ExitCode
  Write-Host "Install exit code: $exitCode"
}

# The installer can return before the ARP entry is written.
$elapsed = 0
while (-not (Get-PaintDotNetEntry) -and ($elapsed -lt $registrationTimeoutSeconds)) {
  Start-Sleep -Seconds 5
  $elapsed += 5
  Write-Host "Waiting for Paint.NET to register... ($elapsed seconds)"
}

Remove-Item -Path $extractPath -Recurse -Force -ErrorAction SilentlyContinue

# In case the installer launches the app once it finishes.
Stop-Process -Name "paintdotnet" -Force -ErrorAction SilentlyContinue

$entry = Get-PaintDotNetEntry
if (-not $entry) {
  Write-Host "Paint.NET did not register in Add/Remove Programs."
  Exit 1
}
Write-Host "Registered '$($entry.DisplayName)' by '$($entry.Publisher)', version $($entry.DisplayVersion)."

# Registration above is the success signal; a killed process's code means nothing.
if ($killed -or $null -eq $exitCode) { Exit 0 }

# 3010 (reboot required) and 1641 (reboot initiated) are successful installs.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
