# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"
$timeoutMinutes = 3

try {

# Evernote uses an electron-builder NSIS installer. It defaults to a per-user
# install, so /allusers is required for a machine-wide install alongside the
# NSIS /S silent flag.

# Setup takes a single-instance mutex and, while another copy holds it, aborts
# with exit code 2 in about two seconds without writing anything. Several of its
# message boxes have no silent-mode default - the one it shows when the previous
# version's uninstaller returns non-zero is MB_OK with no /SD - and nobody can
# answer them from Fleet's SYSTEM context, so that setup stays alive and every
# later install on the host fails the same way until the process is gone. Stop
# setup copies an earlier run left behind. Evernote's own files carry the same
# version stamp, so match only images outside an install directory.
$installDirs = @(Get-ChildItem -Path @(
        'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
        'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
    ) -ErrorAction SilentlyContinue |
    ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
    Where-Object { $_.DisplayName -like 'Evernote*' -and $_.InstallLocation } |
    ForEach-Object { $_.InstallLocation.TrimEnd('\') })
$programFiles = if (${env:ProgramFiles(x86)}) { ${env:ProgramFiles(x86)} } else { ${env:ProgramFiles} }
$installDirs += (Join-Path $programFiles 'Evernote')
$installDirs = @($installDirs | Select-Object -Unique)

$getStrandedSetup = {
    @(Get-Process -ErrorAction SilentlyContinue | Where-Object {
        $procPath = $_.Path
        if (-not $procPath) { return $false }
        $versionInfo = (Get-Item $procPath -ErrorAction SilentlyContinue).VersionInfo
        if ($versionInfo.ProductName -ne 'Evernote') { return $false }
        foreach ($dir in $installDirs) {
            if ($procPath.StartsWith("$dir\", [System.StringComparison]::OrdinalIgnoreCase)) { return $false }
        }
        return $true
    })
}
$stranded = & $getStrandedSetup
foreach ($proc in $stranded) {
    Write-Host "Stopping Evernote setup left behind by an earlier install (PID $($proc.Id))."
    & taskkill.exe /PID $proc.Id /T /F | Out-Null
}
$stranded | Wait-Process -Timeout 30 -ErrorAction SilentlyContinue

# Wait-Process reports a timeout as a non-terminating error, so confirm they
# really stopped rather than letting setup abort on the mutex again.
$stillRunning = & $getStrandedSetup
if ($stillRunning.Count) {
    Write-Host "Error: Evernote setup from an earlier install is still running ($(($stillRunning | ForEach-Object { "$($_.Name) ($($_.Id))" }) -join ', ')). Restart the host and try again."
    Exit 1
}

# The NSIS self-extractor intermittently dies with 0xc0000005 (access
# violation, exit code -1073741819) before extracting anything. The same
# installer succeeds on the next run, so retry on that specific exit code
# only; every other exit code is reported as-is.
$maxAttempts = 3
$exitCode = 1

for ($attempt = 1; $attempt -le $maxAttempts; $attempt++) {
  $process = Start-Process -FilePath "$exeFilePath" -ArgumentList "/allusers /S" -PassThru
  # Without a cached handle, ExitCode is empty once the process has exited.
  $null = $process.Handle

  # Bound the run: setup stopped on one of its own message boxes would otherwise
  # hold the mutex for good and break every later install on this host.
  if (-not $process.WaitForExit($timeoutMinutes * 60 * 1000)) {
    Write-Host "The installer didn't finish within $timeoutMinutes minutes."
    & taskkill.exe /PID $process.Id /T /F | Out-Null
    Exit 1
  }

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
