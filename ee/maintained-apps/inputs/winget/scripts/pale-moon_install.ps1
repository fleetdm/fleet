# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Pale Moon ships as a Nullsoft (NSIS) installer; /S runs it silently.

$exeFilePath = "${env:INSTALLER_PATH}"

try {
    if (-not (Test-Path $exeFilePath)) {
        Write-Host "Error: Installer file not found at: $exeFilePath"
        Exit 1
    }

    # The installer refuses hosts without AVX, including Windows on Arm, with a
    # message box that hangs under SYSTEM. Fail fast instead.
    $nativeArch = (Get-ItemProperty 'HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\Environment').PROCESSOR_ARCHITECTURE
    if ($nativeArch -eq 'ARM64') {
        Write-Host "Error: Pale Moon can't be installed on Windows on Arm. Its installer requires an x64 processor with AVX support."
        Exit 1
    }
    Add-Type -Namespace Fleet -Name Cpu -MemberDefinition '[DllImport("kernel32.dll")] public static extern bool IsProcessorFeaturePresent(uint feature);'
    $PF_AVX_INSTRUCTIONS_AVAILABLE = 39
    if (-not [Fleet.Cpu]::IsProcessorFeaturePresent($PF_AVX_INSTRUCTIONS_AVAILABLE)) {
        Write-Host "Error: Pale Moon can't be installed on this host. Its installer requires a processor with AVX support."
        Exit 1
    }

    $processOptions = @{
        FilePath = "$exeFilePath"
        ArgumentList = "/S"
        PassThru = $true
        Wait = $true
        NoNewWindow = $true
    }

    Write-Host "Starting Pale Moon install with: $($processOptions.ArgumentList)"
    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
