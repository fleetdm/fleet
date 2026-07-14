# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

$stdoutLog = Join-Path $env:TEMP "docker-install-stdout.log"
$stderrLog = Join-Path $env:TEMP "docker-install-stderr.log"

function Write-InstallerLogs {
    $logCandidates = @(
        $stdoutLog,
        $stderrLog,
        "$env:LOCALAPPDATA\Docker\install-log*.txt",
        "$env:ProgramData\DockerDesktop\install-log*.txt"
    )
    foreach ($pattern in $logCandidates) {
        Get-ChildItem -Path $pattern -ErrorAction SilentlyContinue | Where-Object { $_.Length -gt 0 } | ForEach-Object {
            Write-Host "--- $($_.FullName) (last 50 lines) ---"
            Get-Content $_.FullName -Tail 50 | ForEach-Object { Write-Host $_ }
        }
    }
}

try {
    # Docker Desktop 4.72+ added a per-user vs all-user install choice for
    # Windows. All-user silent installs hang on Windows Server runners
    # (Docker Desktop is not officially supported on Windows Server). Install
    # per-user with --user instead: no admin needed, target is
    # %LOCALAPPDATA%\Programs\DockerDesktop, and uninstall info is written to
    # HKCU. --accept-license suppresses the subscription agreement prompt.
    $process = Start-Process -FilePath "$exeFilePath" -ArgumentList "install","--user","--accept-license","--quiet" -PassThru -RedirectStandardOutput $stdoutLog -RedirectStandardError $stderrLog
    # Cache the process handle; without this, .ExitCode is $null once the
    # process exits.
    $null = $process.Handle

    # The installer process can keep running for several minutes after the
    # app is registered with Windows. Poll the HKCU uninstall key (osquery's
    # programs table reads both HKLM and HKU) to detect when the core install
    # has completed, rather than blocking on Start-Process -Wait.
    $registryKey = "HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\Docker Desktop"
    $deadline = (Get-Date).AddMinutes(8)
    while ((Get-Date) -lt $deadline) {
        if (Get-ItemProperty -Path $registryKey -ErrorAction SilentlyContinue) {
            Write-Host "Docker Desktop registered in HKCU."
            Exit 0
        }
        # If the installer already exited without registering the app, fail
        # fast with its exit code and surface the installer logs.
        if ($process.HasExited -and $process.ExitCode -ne 0) {
            Write-Host "Installer exited with code $($process.ExitCode) before registering."
            Write-InstallerLogs
            Exit 1
        }
        Start-Sleep -Seconds 10
    }

    Write-Host "Docker Desktop did not register within timeout."
    Write-InstallerLogs
    Exit 1
} catch {
    Write-Host "Error: $_"
    Exit 1
}
