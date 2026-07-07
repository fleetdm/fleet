# Uninstalls Adobe AIR (HARMAN).
#
# The inner MSI sets ARPSYSTEMCOMPONENT, so the visible ARP entry is the
# vendor-written key "Adobe AIR" whose UninstallString runs
# "...\Adobe AIR Updater.exe" -arp:uninstall. We take that command from the
# registry and add -silent to keep it quiet.

$softwareName = "Adobe AIR"

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

$exitCode = $null

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path @($machineKey, $machineKey32on64) `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath }

foreach ($key in $uninstallKeys) {
    if ($key.DisplayName -eq $softwareName) {
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

        if ($exeArgs -notmatch '(?i)-silent') { $exeArgs = "$exeArgs -silent".Trim() }

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
    Write-Host "Uninstall entry not found for '$softwareName'."
    Exit 1
}

Write-Host "Uninstall exit code: $exitCode"
Exit $exitCode
