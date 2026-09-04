# Uninstalls AnyDesk by invoking the installed AnyDesk.exe with --remove.
# The registry UninstallString runs --uninstall, which opens a GUI prompt;
# --remove --silent is the unattended equivalent.

$softwareName = "AnyDesk"

$machineKey       = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

$exitCode = $null

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path @($machineKey, $machineKey32on64) `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath }

foreach ($key in $uninstallKeys) {
    if ($key.DisplayName -ne $softwareName) { continue }

    # Defensive parser for the three UninstallString shapes:
    #   "C:\path with spaces\AnyDesk.exe" --uninstall  -> quoted
    #   C:\Program Files (x86)\AnyDesk\AnyDesk.exe ... -> unquoted with spaces
    #   MsiExec.exe /X{GUID}                           -> bare token
    $uninstallCommand = $key.UninstallString
    $anyDeskExe = $null
    if ($uninstallCommand -match '^\s*"([^"]+)"') {
        $anyDeskExe = $matches[1]
    } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)') {
        $anyDeskExe = $matches[1]
    } elseif ($uninstallCommand -match '^\s*(\S+)') {
        $anyDeskExe = $matches[1]
    }

    if (-not $anyDeskExe -or -not (Test-Path $anyDeskExe)) {
        Write-Host "AnyDesk executable not found at: $anyDeskExe"
        Exit 1
    }

    Write-Host "Uninstall command: $anyDeskExe --remove --silent"
    $process = Start-Process -FilePath $anyDeskExe -ArgumentList "--remove --silent" -PassThru -Wait
    $exitCode = $process.ExitCode
    break
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
