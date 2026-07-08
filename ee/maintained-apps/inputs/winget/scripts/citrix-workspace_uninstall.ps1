# Uninstalls Citrix Workspace app.
#
# The ARP entry (key CitrixOnlinePluginPackWeb) has a versioned DisplayName
# ("Citrix Workspace 2603") and an UninstallString that runs Citrix's
# TrolleyExpress.exe / CitrixWorkspaceApp.exe. Citrix documents
# "/uninstall /cleanup /silent" as the unattended removal; 3010 (reboot
# required) is a normal success code.

$softwareNameLike = "Citrix Workspace [0-9]*"

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

$exitCode = $null

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path @($machineKey, $machineKey32on64) `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath }

foreach ($key in $uninstallKeys) {
    if ($key.DisplayName -like $softwareNameLike -and $key.Publisher -like "Citrix*") {
        $raw = $key.QuietUninstallString
        if (-not $raw) { $raw = $key.UninstallString }
        if (-not $raw) { continue }

        # Parse into executable + args, handling quoted/unquoted/bare shapes.
        if ($raw -match '^\s*"([^"]+)"\s*(.*)$') {
            $exe = $matches[1]; $exeArgs = $matches[2].Trim()
        } elseif ($raw -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
            $exe = $matches[1]; $exeArgs = $matches[2].Trim()
        } else {
            $exe = $raw; $exeArgs = ""
        }

        if ($exeArgs -notmatch '(?i)/uninstall') { $exeArgs = "/uninstall $exeArgs" }
        if ($exeArgs -notmatch '(?i)/cleanup')   { $exeArgs = "$exeArgs /cleanup" }
        if ($exeArgs -notmatch '(?i)/silent')    { $exeArgs = "$exeArgs /silent" }
        $exeArgs = $exeArgs.Trim()

        Write-Host "Uninstall command: $exe"
        Write-Host "Uninstall args: $exeArgs"
        $process = Start-Process -FilePath $exe -ArgumentList $exeArgs -NoNewWindow -PassThru -Wait
        $exitCode = $process.ExitCode
        break
    }
}

} catch {
    Write-Host "Error: $_"
    Exit 1
}

if ($null -eq $exitCode) {
    Write-Host "Uninstall entry not found matching '$softwareNameLike'."
    Exit 1
}

Write-Host "Uninstall exit code: $exitCode"
# 0 = success, 3010 = success but reboot required, 1641 = reboot initiated
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }
Exit $exitCode
