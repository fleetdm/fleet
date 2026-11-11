# Install Vivaldi silently, machine-wide (Chromium-based browser).
# Fleet runs installs as SYSTEM, so --system-level is required to install for
# all users under %ProgramFiles% (and register under HKLM). Without it the
# installer lands in the SYSTEM profile and is invisible to the real user.

$exeFilePath = "${env:INSTALLER_PATH}"

function Get-ProductName($path) {
  try { [Diagnostics.FileVersionInfo]::GetVersionInfo($path).ProductName } catch { $null }
}

try {

# While Vivaldi is open, its installer asks the user to close it in a dialog
# that a SYSTEM install can't show, so the install never finishes. Until that
# installer is stopped, every later Vivaldi install and uninstall fails (exit
# code 60). Stop Vivaldi and the helpers it runs from its install directory,
# along with any Vivaldi installer left running by an earlier attempt. Unsaved
# changes are lost.
$installDirs = @("${env:ProgramFiles}\Vivaldi\Application\") + @(
  Get-ItemProperty @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
  ) -ErrorAction SilentlyContinue |
    Where-Object { $_.DisplayName -eq 'Vivaldi' -and $_.Publisher -eq 'Vivaldi Technologies AS.' -and $_.InstallLocation } |
    ForEach-Object { $_.InstallLocation.Trim('"').TrimEnd('\') + '\' }
)
$running = @(Get-CimInstance Win32_Process -ErrorAction Stop | Where-Object {
  $path = $_.ExecutablePath
  $path -and (
    ($installDirs | Where-Object { $path.StartsWith($_, [StringComparison]::OrdinalIgnoreCase) }) -or
    ($_.CommandLine -match '\s--system-level\b' -and (Get-ProductName $path) -eq 'Vivaldi Installer')
  )
})
if ($running) {
  Write-Host "Stopping $(($running.Name | Sort-Object -Unique) -join ', ') ($($running.Count) processes) before installing."
  $stopped = @{}
  foreach ($proc in $running) {
    $stopped["$($proc.ProcessId)|$($proc.CreationDate.Ticks)"] = $true
    Stop-Process -Id $proc.ProcessId -Force -ErrorAction SilentlyContinue
  }
  # Wait-Process can return while stopped processes are still exiting, so poll.
  # Match on start time too, since Windows reuses PIDs.
  $sw = [Diagnostics.Stopwatch]::StartNew()
  do {
    Start-Sleep -Milliseconds 500
    $survivors = @(Get-CimInstance Win32_Process -ErrorAction Stop |
      Where-Object { $stopped["$($_.ProcessId)|$($_.CreationDate.Ticks)"] })
  } while ($survivors -and $sw.Elapsed.TotalSeconds -lt 30)
  if ($survivors) {
    Write-Host "Couldn't stop $(($survivors.Name | Sort-Object -Unique) -join ', ') (PID $($survivors.ProcessId -join ', ')), so not installing."
    Exit 1
  }
}

$process = Start-Process -FilePath $exeFilePath `
  -ArgumentList "--vivaldi-silent --do-not-launch-chrome --system-level" `
  -NoNewWindow -PassThru -Wait
Write-Host "Install exit code: $($process.ExitCode)"
Exit $process.ExitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
