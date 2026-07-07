# Uninstalls the 3DxWare 10 driver suite.
#
# 3DxWare registers a WiX burn bundle ARP entry (DisplayName starts with
# "3Dconnexion 3DxWare 10"; older builds append a suffix like "(64-bit)").
# Burn bundles uninstall by running the cached bootstrapper .exe with
# /uninstall /quiet /norestart -- never via msiexec.

$softwareNameLike = "3Dconnexion 3DxWare 10*"

function Invoke-Uninstaller {
    param([string]$exe, [string]$exeArgs)
    if ($exeArgs -notmatch '/uninstall') { $exeArgs = "/uninstall $exeArgs" }
    if ($exeArgs -notmatch '/quiet')     { $exeArgs = "$exeArgs /quiet" }
    if ($exeArgs -notmatch '/norestart') { $exeArgs = "$exeArgs /norestart" }
    $exeArgs = $exeArgs.Trim()
    Write-Host "Uninstall command: $exe"
    Write-Host "Uninstall args: $exeArgs"
    $process = Start-Process -FilePath $exe -ArgumentList $exeArgs -NoNewWindow -PassThru -Wait
    return $process.ExitCode
}

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

$exitCode = $null

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path @($machineKey, $machineKey32on64) `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath }

foreach ($key in $uninstallKeys) {
    if ($key.DisplayName -like $softwareNameLike) {
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

        $exitCode = Invoke-Uninstaller -exe $exe -exeArgs $exeArgs
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
