# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# /allusers installs Stretchly machine-wide. Without it, the installer puts the
# app in the SYSTEM account's profile, where no one can open it. The installer
# removes a copy an earlier install left there; if that copy's uninstaller is
# missing, the copy is removed here first.

$exeFilePath = "${env:INSTALLER_PATH}"
$legacyKey = 'Registry::HKEY_USERS\S-1-5-18\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\4f357cc2-fcfb-5fb1-89af-67446d29eff0'
$legacyDir = "$env:SystemRoot\SysWOW64\config\systemprofile\AppData\Local\Programs\Stretchly"

try {
    if (-not (Test-Path $exeFilePath)) {
        Write-Host "Error: Installer file not found at: $exeFilePath"
        Exit 1
    }

    $legacy = Get-ItemProperty -LiteralPath $legacyKey -ErrorAction SilentlyContinue
    if ($legacy -and $legacy.UninstallString -match '^\s*"([^"]+)"') {
        # The 32-bit installer wrote it under SysWOW64 but registered the system32 path.
        $legacyExe = $matches[1] -replace '(?i)\\system32\\', '\SysWOW64\'
        if ((Split-Path $legacyExe -Parent) -eq $legacyDir -and -not (Test-Path -LiteralPath $legacyExe -PathType Leaf)) {
            Write-Host "Removing the copy of Stretchly in $legacyDir"
            Remove-Item -LiteralPath $legacyDir -Recurse -Force -ErrorAction SilentlyContinue
            if (-not (Test-Path -LiteralPath $legacyDir)) {
                Remove-Item -LiteralPath $legacyKey -Recurse -Force -ErrorAction SilentlyContinue
            }
        }
    }

    $processOptions = @{
        FilePath = "$exeFilePath"
        ArgumentList = "/S /allusers"
        PassThru = $true
        Wait = $true
        NoNewWindow = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
