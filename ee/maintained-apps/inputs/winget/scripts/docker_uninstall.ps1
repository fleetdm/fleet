# Define acceptable/expected exit codes
$ExpectedExitCodes = @(0, 19)

# Docker Desktop can be installed per-user (HKCU, %LOCALAPPDATA%) or
# all-users (HKLM, C:\Program Files). Check HKCU first, then HKLM.
$registryPaths = @(
    'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\Docker Desktop',
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\Docker Desktop'
)

# Initialize exit code
$exitCode = 0

try {
    $key = $null
    foreach ($path in $registryPaths) {
        $candidate = Get-ItemProperty -Path $path -ErrorAction SilentlyContinue
        if ($candidate) {
            Write-Host "Found Docker Desktop registry at: $path"
            $key = $candidate
            break
        }
    }
    if (-not $key) {
        Throw "Docker Desktop registry entry not found in HKCU or HKLM."
    }

    # Get the uninstall command. Some uninstallers do not include 'QuietUninstallString'
    $uninstallCommand = if ($key.QuietUninstallString) {
        $key.QuietUninstallString
    } else {
        $key.UninstallString
    }

    # The expected uninstall command value is "<install dir>\Docker Desktop Installer.exe" "uninstall"
    $splitArgs = $uninstallCommand.Split('"')
    if ($splitArgs.Length -ne 5) {
      Throw "Unexpected uninstall command. Please update the uninstall script.`nUninstall command: $uninstallCommand"
    }
    $uninstallCommand = $splitArgs[1]
    $uninstallArgs = $splitArgs[3]

    Write-Host "Uninstall command: $uninstallCommand"
    Write-Host "Uninstall args: $uninstallArgs"

    $processOptions = @{
        FilePath = $uninstallCommand
        PassThru = $true
        Wait     = $true
    }
    if ($uninstallArgs -ne '') {
        $processOptions.ArgumentList = "$uninstallArgs --quiet"
    } else {
        $processOptions.ArgumentList = "--quiet"
    }

    # Start uninstall process
    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Uninstall exit code: $exitCode"

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
}

# Treat acceptable exit codes as success
if ($ExpectedExitCodes -contains $exitCode) {
    Exit 0
} else {
    Exit $exitCode
}
