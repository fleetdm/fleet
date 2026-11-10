# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

# The installer defaults to the running user's %LOCALAPPDATA%, which is the
# SYSTEM profile when Fleet installs it, so pass a machine-wide location.
$installDir = Join-Path $env:ProgramFiles "Scrivener3"

try {
    if (-not (Test-Path $exeFilePath)) {
        Write-Host "Error: Installer file not found at: $exeFilePath"
        Exit 1
    }

    # Earlier installs landed in the SYSTEM profile. The new install takes over
    # the registration and shortcuts, so only the old files need removing.
    $strandedDir = $null
    $existing = Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*' -ErrorAction SilentlyContinue |
        Where-Object { $_.DisplayName -eq 'Scrivener' -and $_.Publisher -like 'Literature and Latte*' } |
        Select-Object -First 1
    if ($existing -and $existing.InstallLocation -match '(?i)\\config\\systemprofile\\') {
        $strandedDir = $existing.InstallLocation
    }

    $processOptions = @{
        FilePath = "$exeFilePath"
        ArgumentList = "--mode unattended --unattendedmodeui none --prefix `"$installDir`""
        PassThru = $true
        Wait = $true
        NoNewWindow = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"

    if ($exitCode -eq 0 -and $strandedDir -and (Test-Path -LiteralPath $strandedDir)) {
        Write-Host "Removing previous install from the SYSTEM profile: $strandedDir"
        Remove-Item -LiteralPath $strandedDir -Recurse -Force -ErrorAction SilentlyContinue
    }

    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
