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
    # Like the installer, only check AVX on Windows 10 2004 (build 19041) and
    # later; older Windows doesn't report it.
    $build = [int](Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion').CurrentBuild
    if ($build -ge 19041) {
        $hasAvx = $true
        try {
            Add-Type -Namespace Fleet -Name Cpu -MemberDefinition '[DllImport("kernel32.dll")] public static extern bool IsProcessorFeaturePresent(uint feature);'
            $PF_AVX_INSTRUCTIONS_AVAILABLE = 39
            $hasAvx = [Fleet.Cpu]::IsProcessorFeaturePresent($PF_AVX_INSTRUCTIONS_AVAILABLE)
        } catch {
            # Add-Type is blocked in ConstrainedLanguage mode; leave the check to the installer.
            Write-Host "Couldn't check for AVX support: $_"
        }
        if (-not $hasAvx) {
            Write-Host "Error: Pale Moon can't be installed on this host. Its installer requires a processor with AVX support."
            Exit 1
        }
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
