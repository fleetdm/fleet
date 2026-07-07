# Uninstalls CutePDF Writer.
#
# CutePDF registers under the key "CutePDF Writer Installation" with
# DisplayName "CutePDF Writer". Its uninstaller is a custom printer-driver
# remover (unInstcpw64.exe /uninstall), not an Inno unins000.exe; /s makes it
# silent.

$softwareName = "CutePDF Writer"

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

        if ($exeArgs -notmatch '(?i)/uninstall') { $exeArgs = "/uninstall $exeArgs".Trim() }
        if ($exeArgs -notmatch '(?i)(^|\s)/s(\s|$)') { $exeArgs = "$exeArgs /s".Trim() }

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
