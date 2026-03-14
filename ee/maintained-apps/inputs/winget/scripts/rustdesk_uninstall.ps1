# Define acceptable/expected exit codes
$ExpectedExitCodes = @(0)

# Look up both machine and user uninstall registry locations
$regPaths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\RustDesk',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\RustDesk'
)

$exitCode = 0

try {
    $key = $null
    foreach ($path in $regPaths) {
        if (Test-Path $path) {
            $key = Get-ItemProperty -Path $path -ErrorAction Stop
            break
        }
    }

    if ($null -eq $key) {
        Write-Host "RustDesk uninstall registry key not found; app may already be uninstalled."
        Exit 0
    }

    # Use QuietUninstallString if available (MSI sets this to include /quiet)
    $uninstallCommand = if ($key.QuietUninstallString) {
        $key.QuietUninstallString
    } else {
        $key.UninstallString
    }

    Write-Host "Uninstall command: $uninstallCommand"

    # MsiExec.exe /X{...} commands should be run directly (not split)
    if ($uninstallCommand -match 'MsiExec\.exe') {
        # Run msiexec uninstall with quiet flag
        $msiArgs = $uninstallCommand -replace '^.*MsiExec\.exe\s*', ''
        if ($msiArgs -notmatch '/quiet') {
            $msiArgs = "$msiArgs /quiet /norestart"
        }
        Write-Host "Running msiexec with args: $msiArgs"
        $process = Start-Process msiexec.exe -ArgumentList $msiArgs -PassThru -Wait
        $exitCode = $process.ExitCode
    } else {
        # Fallback: split quoted command and args
        $splitArgs = $uninstallCommand.Split('"')
        $uninstallExe = $splitArgs[1]
        $uninstallArgs = if ($splitArgs.Length -eq 3) { $splitArgs[2].Trim() } else { '' }

        $processOptions = @{
            FilePath = $uninstallExe
            PassThru = $true
            Wait     = $true
        }
        if ($uninstallArgs -ne '') {
            $processOptions.ArgumentList = $uninstallArgs
        }

        $process = Start-Process @processOptions
        $exitCode = $process.ExitCode
    }

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
