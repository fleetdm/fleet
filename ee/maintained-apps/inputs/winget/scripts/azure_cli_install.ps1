# Learn more about .msi install scripts:
# http://fleetdm.com/learn-more-about/msi-install-scripts
#
# Installing this MSI over an identical version puts Windows Installer into
# maintenance mode, which reconfigures the product and fails with 1603 instead
# of succeeding. Compare the installed version against the MSI's first and skip
# the install when the target version is already in place.

$msiFilePath = "${env:INSTALLER_PATH}"
$logFile = "${env:TEMP}/fleet-install-software.log"

# 3010 = ERROR_SUCCESS_REBOOT_REQUIRED, 1641 = ERROR_SUCCESS_REBOOT_INITIATED.
$successCodes = @(0, 3010, 1641)

function Get-MsiProductVersion {
    param([string]$Path)
    $installer = New-Object -ComObject WindowsInstaller.Installer
    try {
        $database = $installer.GetType().InvokeMember(
            'OpenDatabase', 'InvokeMethod', $null, $installer, @($Path, 0))
        $view = $database.GetType().InvokeMember(
            'OpenView', 'InvokeMethod', $null, $database,
            @("SELECT Value FROM Property WHERE Property = 'ProductVersion'"))
        $view.GetType().InvokeMember('Execute', 'InvokeMethod', $null, $view, $null)
        $record = $view.GetType().InvokeMember('Fetch', 'InvokeMethod', $null, $view, $null)
        if ($record) {
            return $record.GetType().InvokeMember(
                'StringData', 'GetProperty', $null, $record, @(1))
        }
    } finally {
        [System.Runtime.InteropServices.Marshal]::FinalReleaseComObject($installer) | Out-Null
    }
    return $null
}

try {

$msiVersion = Get-MsiProductVersion -Path $msiFilePath
$installed = Get-ItemProperty `
    -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*' `
    -ErrorAction SilentlyContinue |
        Where-Object { $_.DisplayName -eq 'Microsoft Azure CLI (64-bit)' } |
        Select-Object -First 1

if ($msiVersion -and $installed -and $installed.DisplayVersion -eq $msiVersion) {
    Write-Host "Microsoft Azure CLI $msiVersion is already installed; nothing to do."
    Exit 0
}

$installProcess = Start-Process msiexec.exe `
  -ArgumentList "/quiet /norestart /lv `"${logFile}`" /i `"${msiFilePath}`"" `
  -PassThru -Verb RunAs -Wait

Get-Content $logFile -Tail 500

if ($successCodes -contains $installProcess.ExitCode) {
  Exit 0
}

Exit $installProcess.ExitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
