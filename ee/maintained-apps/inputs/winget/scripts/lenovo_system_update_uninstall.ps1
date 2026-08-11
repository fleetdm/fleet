# Fleet substitutes the winget ProductCode, which for this Inno installer is the
# uninstall registry key name rather than a GUID.
$packageId = $PACKAGE_ID
$uninstallArgs = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART"

# The installer is x86, so on 64-bit Windows it registers under Wow6432Node.
$paths = @(
    "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\$packageId",
    "HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\$packageId"
)
$exitCode = 0

try {
    $key = $paths |
        ForEach-Object { Get-ItemProperty -Path $_ -ErrorAction SilentlyContinue } |
        Select-Object -First 1

    if (-not $key) { Write-Host "Uninstall entry not found for '$packageId'."; Exit 0 }

    $uninstallCommand = if ($key.QuietUninstallString) { $key.QuietUninstallString } else { $key.UninstallString }
    if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
        $uninstallCommand = $Matches[1]; if ($Matches[2]) { $uninstallArgs = "$($Matches[2]) $uninstallArgs".Trim() }
    } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
        $uninstallCommand = $Matches[1]; if ($Matches[2]) { $uninstallArgs = "$($Matches[2]) $uninstallArgs".Trim() }
    } elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
        $uninstallCommand = $Matches[1]; if ($Matches[2]) { $uninstallArgs = "$($Matches[2]) $uninstallArgs".Trim() }
    }

    Write-Host "Uninstall command: $uninstallCommand"; Write-Host "Uninstall args: $uninstallArgs"
    $processOptions = @{ FilePath = $uninstallCommand; PassThru = $true; Wait = $true }
    if ($uninstallArgs -ne '') { $processOptions.ArgumentList = $uninstallArgs }
    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode; Write-Host "Uninstall exit code: $exitCode"
} catch { Write-Host "Error: $_"; Exit 1 }

Exit $exitCode
