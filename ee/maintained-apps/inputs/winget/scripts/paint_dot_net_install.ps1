# Paint.NET ships as a zip containing its installer .exe.

$zipFilePath = "${env:INSTALLER_PATH}"

$installTimeoutSeconds = 420
$registrationTimeoutSeconds = 120

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

# Same DisplayName and Publisher the catalog's exists query uses.
function Get-PaintDotNetEntry {
    Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -eq "Paint.NET" -and $_.Publisher -eq "dotPDN LLC" } |
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

# /auto is the vendor's silent switch. -Wait would also wait on descendants.
$process = Start-Process -FilePath $installer.FullName -ArgumentList "/auto" -PassThru
$null = $process.Handle  # keeps .ExitCode readable after exit

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

Stop-Process -Name "paintdotnet" -Force -ErrorAction SilentlyContinue

$entry = Get-PaintDotNetEntry
if (-not $entry) {
  Write-Host "Paint.NET did not register in Add/Remove Programs."
  Exit 1
}
Write-Host "Registered '$($entry.DisplayName)' by '$($entry.Publisher)', version $($entry.DisplayVersion)."

# Registration is the success signal; a killed process's code means nothing.
if ($killed -or $null -eq $exitCode) { Exit 0 }

# 3010/1641 = reboot required/initiated.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
