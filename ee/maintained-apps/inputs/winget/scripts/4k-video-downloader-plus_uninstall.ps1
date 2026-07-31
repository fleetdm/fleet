# Uninstalls 4K Video Downloader+.
#
# The app is a WiX "burn" bundle that chains an MSI, and the ARP entry with
# DisplayName "4K Video Downloader+" may be either the bundle (an .exe
# UninstallString, needs /uninstall /quiet /norestart) or the chained MSI
# (an MsiExec.exe /X{ProductCode} UninstallString, needs /qn /norestart --
# NOT /uninstall, which is invalid for msiexec). Handle both shapes. The "+"
# in the exact-match name keeps this from touching the separate MSI-based
# "4K Video Downloader" product.

$softwareName = "4K Video Downloader+"

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

function Split-UninstallString {
    param([string]$raw)
    # Parse into executable + args, handling quoted/unquoted/bare shapes.
    if ($raw -match '^\s*"([^"]+)"\s*(.*)$') {
        return @($matches[1], $matches[2].Trim())
    } elseif ($raw -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
        return @($matches[1], $matches[2].Trim())
    }
    return @($raw, "")
}

$exitCode = $null

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path @($machineKey, $machineKey32on64) `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath }

[array]$entries = $uninstallKeys | Where-Object { $_.DisplayName -eq $softwareName }

# Prefer the burn bundle entry (non-msiexec .exe uninstaller): it removes the
# whole chain, including the MSI. Fall back to the chained MSI entry.
$bundle = $entries | Where-Object {
    $raw = if ($_.QuietUninstallString) { $_.QuietUninstallString } else { $_.UninstallString }
    $raw -and $raw -notmatch '(?i)msiexec'
} | Select-Object -First 1
$entry = if ($bundle) { $bundle } else { $entries | Select-Object -First 1 }

if ($entry) {
    $raw = if ($entry.QuietUninstallString) { $entry.QuietUninstallString } else { $entry.UninstallString }
    $exe, $exeArgs = Split-UninstallString -raw $raw

    if ($exe -match '(?i)msiexec') {
        if ($exeArgs -notmatch '(?i)/(x|uninstall)') { $exeArgs = "/X $exeArgs" }
        if ($exeArgs -notmatch '(?i)/(qn|quiet)') { $exeArgs = "$exeArgs /qn" }
        if ($exeArgs -notmatch '(?i)/norestart') { $exeArgs = "$exeArgs /norestart" }
    } else {
        if ($exeArgs -notmatch '/uninstall') { $exeArgs = "/uninstall $exeArgs" }
        if ($exeArgs -notmatch '/quiet')     { $exeArgs = "$exeArgs /quiet" }
        if ($exeArgs -notmatch '/norestart') { $exeArgs = "$exeArgs /norestart" }
    }
    $exeArgs = $exeArgs.Trim()

    Write-Host "Uninstall command: $exe"
    Write-Host "Uninstall args: $exeArgs"
    $process = Start-Process -FilePath $exe -ArgumentList $exeArgs -NoNewWindow -PassThru -Wait
    $exitCode = $process.ExitCode
}

} catch {
    Write-Host "Error: $_"
    Exit 1
}

if ($null -eq $exitCode) {
    Write-Host "Uninstall entry not found for '$softwareName'."
    Exit 1
}

Write-Host "Uninstall exit code: $exitCode"
# 0 = success, 3010 = success but reboot required, 1641 = reboot initiated
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }
Exit $exitCode
