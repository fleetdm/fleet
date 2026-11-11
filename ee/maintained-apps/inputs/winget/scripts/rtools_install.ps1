# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Rtools won't install into an existing folder (run silently, it waits on a
# hidden error dialog), so an earlier install of the same release in
# C:\rtools<version>, including MSYS2 home folders under it, is removed first.

$exeFilePath = "${env:INSTALLER_PATH}"

# Extracting ~52,000 files can take well over 10 minutes with antivirus
# scanning. This stays under Fleet's 1-hour script limit.
$timeoutSeconds = 3300
$pollSeconds = 15
$installArgs = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART"

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

function Get-RtoolsKeys {
    Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -like "Rtools*" }
}

function Get-RtoolsEntries {
    Get-RtoolsKeys | ForEach-Object { "$($_.PSChildName)|$($_.DisplayVersion)|$($_.InstallLocation)" }
}

function Test-RtoolsRegistered {
    $null -ne (Get-RtoolsKeys | Select-Object -First 1)
}

function Stop-ProcessTree($processId) {
    Get-CimInstance Win32_Process -Filter "ParentProcessId=$processId" -ErrorAction SilentlyContinue |
        ForEach-Object { Stop-ProcessTree $_.ProcessId }
    Stop-Process -Id $processId -Force -ErrorAction SilentlyContinue
}

# Waits on the process alone: -Wait would also wait on processes it leaves running.
function Wait-ProcessUntil($process, $stopwatch) {
    while (-not $process.HasExited -and ($stopwatch.Elapsed.TotalSeconds -lt $timeoutSeconds)) {
        Start-Sleep -Seconds $pollSeconds
    }
    $process.HasExited
}

try {

$stopwatch = [System.Diagnostics.Stopwatch]::StartNew()

# The installer's default folder is C:\rtools<major><minor>, e.g. C:\rtools45.
$installerInfo = (Get-Item $exeFilePath).VersionInfo
$installDir = "C:\rtools$($installerInfo.FileMajorPart)$($installerInfo.FileMinorPart)"

foreach ($key in @(Get-RtoolsKeys | Where-Object { $_.InstallLocation -and $_.InstallLocation.TrimEnd('\') -eq $installDir })) {
  $uninstallCommand = if ($key.QuietUninstallString) { $key.QuietUninstallString } else { $key.UninstallString }
  if ($uninstallCommand -match '^\s*"([^"]+)"') {
    $uninstaller = $Matches[1]
  } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)') {
    $uninstaller = $Matches[1]
  } else {
    continue
  }
  Write-Host "Removing $($key.DisplayName) from $installDir before installing."
  $uninstall = Start-Process -FilePath $uninstaller -ArgumentList $installArgs -PassThru
  $null = $uninstall.Handle
  if (-not (Wait-ProcessUntil $uninstall $stopwatch)) {
    Write-Host "Uninstalling $($key.DisplayName) did not finish within ${timeoutSeconds}s."
    Stop-ProcessTree $uninstall.Id
    Exit 1
  }
  Write-Host "Uninstall exit code: $($uninstall.ExitCode)"
  # The uninstaller hands off to a copy of itself in a temp folder.
  while ((Test-Path -LiteralPath $key.PSPath) -and ($stopwatch.Elapsed.TotalSeconds -lt $timeoutSeconds)) {
    Start-Sleep -Seconds $pollSeconds
  }
}

if (Test-Path -LiteralPath $installDir) {
  Write-Host "Removing leftover folder $installDir before installing."
  Get-Process -ErrorAction SilentlyContinue |
    Where-Object { $_.Path -like "$installDir\*" } |
    Stop-Process -Force -ErrorAction SilentlyContinue
  # rmdir removes junctions in the folder without following them.
  cmd.exe /c rmdir /s /q $installDir
  if (Test-Path -LiteralPath $installDir) {
    Write-Host "$installDir already exists and couldn't be removed. Rtools can't install into an existing folder."
    Exit 1
  }
}

# An Rtools entry left by an earlier install doesn't mean this one finished.
$preexistingEntries = @(Get-RtoolsEntries)

function Test-RtoolsRegisteredByThisInstall {
    $null -ne (Get-RtoolsEntries | Where-Object { $preexistingEntries -notcontains $_ } | Select-Object -First 1)
}

$process = Start-Process -FilePath "$exeFilePath" -ArgumentList $installArgs -PassThru
# Keeps .ExitCode readable after the process ends.
$null = $process.Handle

$nextLogSeconds = [int]$stopwatch.Elapsed.TotalSeconds + 60
while (-not $process.HasExited -and ($stopwatch.Elapsed.TotalSeconds -lt $timeoutSeconds)) {
  Start-Sleep -Seconds $pollSeconds
  $elapsed = [int]$stopwatch.Elapsed.TotalSeconds
  if ($elapsed -ge $nextLogSeconds) {
    Write-Host "Installing... ($elapsed seconds, registered: $(Test-RtoolsRegisteredByThisInstall))"
    $nextLogSeconds = $elapsed + 60
  }
}
$elapsed = [int]$stopwatch.Elapsed.TotalSeconds

if (-not $process.HasExited) {
  # Registered means the files are in place and only a lingering process remains.
  if (Test-RtoolsRegisteredByThisInstall) {
    Write-Host "Installer still running after ${elapsed}s but Rtools is registered; stopping the lingering process."
    Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
    Start-Sleep -Seconds 2
    Exit 0
  }

  Write-Host "Installer did not finish within ${elapsed}s and Rtools is not registered."
  Stop-ProcessTree $process.Id
  Exit 1
}

$exitCode = $process.ExitCode
Write-Host "Install exit code: $exitCode after $elapsed seconds"

# The parent can exit while a descendant is still writing the ARP entry.
$settle = 0
while (-not (Test-RtoolsRegistered) -and ($settle -lt 90)) {
  Start-Sleep -Seconds $pollSeconds
  $settle += $pollSeconds
  Write-Host "Waiting for Rtools to register... ($settle seconds)"
}

if (-not (Test-RtoolsRegistered)) {
  Write-Host "Rtools did not register in Add/Remove Programs."
  Exit 1
}

# 3010 (reboot required) and 1641 (reboot initiated) are successful installs.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
