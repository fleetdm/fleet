# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

# TreeSize Free uses Inno Setup. The switches below are the machine-scope
# "Silent" switches from the winget manifest; /SP- additionally suppresses the
# "This will install..." prompt.
#
# The install stalls before it starts. Inno's own log shows why:
#
#   Extracting temporary file: ...\is-XXXXX.tmp\TreeSizeFree.exe
#   (seven minutes pass)
#   InitializeSetup returned False; aborting.
#
# InitializeSetup extracts TreeSizeFree.exe and runs it, then blocks waiting for
# it to exit. On a machine with no interactive desktop that child never exits, so
# InitializeSetup eventually times out and aborts the whole install -- which is
# why nothing ever landed in Program Files and nothing was ever registered.
#
# Killing that child does not help -- InitializeSetup needs it to *succeed*, so a
# killed child aborts the install just as fast. What it is stuck on is a window
# waiting for input, so close the window instead and let the child exit cleanly.
# (The same approach fixed the GnuPG and Gpg4win installers in this batch.) Child
# window titles are logged, so if this still aborts the CI log will say what the
# dialog was.
#
# Waiting on the installer process alone matters regardless: "Start-Process -Wait"
# waits for the process *and all of its descendants*. Inno's /LOG is kept so any
# future stall is diagnosable straight from the CI log.
$installTimeoutSeconds = 420
$pollSeconds = 15
$graceSeconds = 20
$leftovers = @("TreeSize", "TreeSizeFree")

$innoLog = Join-Path $env:TEMP "treesize-free-install.log"

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
$userKey = 'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'

# Collect the installer process and everything descended from it, so a dialog
# owned by a child (rather than the installer itself) is still visible to us.
function Get-InstallerTree([int]$rootId) {
    $all = @{}
    Get-CimInstance Win32_Process -ErrorAction SilentlyContinue |
        ForEach-Object { $all[[int]$_.ProcessId] = [int]$_.ParentProcessId }

    $ids = New-Object System.Collections.Generic.HashSet[int]
    $null = $ids.Add($rootId)
    # Walk down a bounded number of generations; the tree here is shallow.
    for ($depth = 0; $depth -lt 5; $depth++) {
        foreach ($procId in @($all.Keys)) {
            if ($ids.Contains($all[$procId])) { $null = $ids.Add($procId) }
        }
    }

    Get-Process -ErrorAction SilentlyContinue | Where-Object { $ids.Contains($_.Id) }
}

function Test-TreeSizeRegistered {
    $null -ne (Get-ChildItem -Path @($machineKey, $machineKey32on64, $userKey) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -like "TreeSize Free*" } |
        Select-Object -First 1)
}

function Write-InnoLog {
    if (Test-Path $innoLog) {
        Write-Host "--- Inno Setup log (last 60 lines) ---"
        Get-Content $innoLog -Tail 60 | ForEach-Object { Write-Host $_ }
        Write-Host "--- end of Inno Setup log ---"
    } else {
        Write-Host "No Inno Setup log was written at $innoLog."
    }
}

try {

$process = Start-Process -FilePath "$exeFilePath" `
  -ArgumentList "/SP- /VERYSILENT /SUPPRESSMSGBOXES /NORESTART /AllUsers /LOG=`"$innoLog`"" `
  -PassThru
# Touch .Handle so the exit code is still readable after the process ends:
# Start-Process -PassThru otherwise returns $null for .ExitCode.
$null = $process.Handle

$elapsed = 0
while (-not $process.HasExited -and ($elapsed -lt $installTimeoutSeconds)) {
  Start-Sleep -Seconds $pollSeconds
  $elapsed += $pollSeconds
  $process.Refresh()
  if ($process.HasExited) { break }

  $tree = Get-InstallerTree $process.Id
  $names = @($tree | Select-Object -ExpandProperty ProcessName -Unique)
  $windows = @($tree | Where-Object { $_.MainWindowHandle -ne [IntPtr]::Zero } |
    ForEach-Object { "$($_.ProcessName): '$($_.MainWindowTitle)'" })

  Write-Host "Installing... ($elapsed seconds, registered: $(Test-TreeSizeRegistered), tree: $($names -join ', '), windows: $($windows -join ' | '))"

  if ($elapsed -ge $graceSeconds) {
    # Release InitializeSetup by dismissing whatever the extracted TreeSizeFree.exe
    # is waiting on, so it can exit on its own. Killing it instead makes
    # InitializeSetup return False and abort the install.
    foreach ($p in $tree) {
      if ($p.MainWindowHandle -ne [IntPtr]::Zero) {
        Write-Host "Closing window owned by $($p.ProcessName) ('$($p.MainWindowTitle)') so the install can continue."
        $null = $p.CloseMainWindow()
      }
    }
  }
}

if (-not $process.HasExited) {
  Write-Host "Installer still running after ${installTimeoutSeconds}s; stopping it."
  Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
  Start-Sleep -Seconds 2
} else {
  Write-Host "Install exit code: $($process.ExitCode)"
}

foreach ($name in $leftovers) { Stop-Process -Name $name -Force -ErrorAction SilentlyContinue }

# Registration is the success signal, not the exit code: on the timeout path the
# installer was killed, so its exit code says nothing about the install.
if (-not (Test-TreeSizeRegistered)) {
  Write-Host "TreeSize Free did not register in Add/Remove Programs."
  Write-InnoLog
  Exit 1
}

Write-Host "TreeSize Free is registered in Add/Remove Programs."
Exit 0

} catch {
  Write-Host "Error: $_"
  Write-InnoLog
  Exit 1
}
