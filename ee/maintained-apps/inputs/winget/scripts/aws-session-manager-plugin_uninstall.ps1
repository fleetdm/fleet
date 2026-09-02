# Uninstalls the AWS Session Manager Plugin.
#
# The plugin is a WiX "burn" bundle that chains an MSI, and both may register
# ARP entries with DisplayName "Session Manager Plugin". Prefer the bundle
# entry (an .exe UninstallString, uninstalls the whole chain with
# /uninstall /quiet /norestart); fall back to the chained MSI entry
# (msiexec /X{ProductCode}) if only that one is present.

$softwareName = "Session Manager Plugin"

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

[array]$matches_ = $uninstallKeys | Where-Object { $_.DisplayName -eq $softwareName }

# Prefer the burn bundle entry (non-msiexec .exe uninstaller) over the chained MSI.
$bundle = $matches_ | Where-Object {
    $raw = if ($_.QuietUninstallString) { $_.QuietUninstallString } else { $_.UninstallString }
    $raw -and $raw -notmatch '(?i)msiexec'
} | Select-Object -First 1
$entry = if ($bundle) { $bundle } else { $matches_ | Select-Object -First 1 }

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
