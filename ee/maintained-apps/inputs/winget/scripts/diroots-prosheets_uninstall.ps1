# Uninstalls DiRoots ProSheets.
#
# ProSheets is an Advanced Installer MSI that registers an ARP entry
# (DisplayName "ProSheets"). Its UninstallString is either an MsiExec /X{code}
# or the AI setup.exe with "/x //"; handle both and force a silent uninstall.
# Note: the install also drops bundled PDF24 Creator + a virtual printer under
# their own ARP entries; those are intentionally left in place (removing them
# breaks a re-install and they are shared components).

$softwareName = "ProSheets"

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

        if ($exe -match '(?i)msiexec') {
            if ($exeArgs -notmatch '(?i)/(x|uninstall)') { $exeArgs = "/X $exeArgs" }
            if ($exeArgs -notmatch '(?i)/(qn|quiet)') { $exeArgs = "$exeArgs /qn" }
            if ($exeArgs -notmatch '(?i)/norestart') { $exeArgs = "$exeArgs /norestart" }
        } else {
            # Advanced Installer setup.exe: "/x //" runs a silent uninstall.
            if ($exeArgs -notmatch '/x') { $exeArgs = "/x $exeArgs" }
            if ($exeArgs -notmatch '//') { $exeArgs = "$exeArgs //" }
            if ($exeArgs -notmatch '(?i)/qn') { $exeArgs = "$exeArgs /qn" }
        }
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
    Write-Host "Uninstall entry not found for '$softwareName'."
    Exit 1
}

Write-Host "Uninstall exit code: $exitCode"
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }
Exit $exitCode
