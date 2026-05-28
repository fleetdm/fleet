# Uninstalls WinSCP (Inno Setup installer).
# Looks up the registry UninstallString and runs it with Inno's silent flags.

$softwareNameLike = "WinSCP*"
$silentArgs = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART"

$machineKey      = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

$exitCode = 0

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path @($machineKey, $machineKey32on64) `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath }

$foundUninstaller = $false
foreach ($key in $uninstallKeys) {
    if ($key.DisplayName -like $softwareNameLike) {
        $foundUninstaller = $true

        # Prefer QuietUninstallString when present; otherwise use UninstallString.
        $uninstallCommand = if ($key.QuietUninstallString) {
            $key.QuietUninstallString
        } else {
            $key.UninstallString
        }

        # Defensive parser for the three UninstallString shapes:
        #   "C:\path with spaces\unins000.exe" /ARG    -> quoted
        #   C:\Program Files\WinSCP\unins000.exe /ARG  -> unquoted with spaces (Inno default)
        #   MsiExec.exe /X{GUID}                       -> bare token
        $uninstallPath = $null
        $existingArgs  = ''
        if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
            $uninstallPath = $matches[1]
            $existingArgs  = $matches[2]
        } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
            $uninstallPath = $matches[1]
            $existingArgs  = $matches[2]
        } elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
            $uninstallPath = $matches[1]
            $existingArgs  = $matches[2]
        }

        $finalArgs = ($existingArgs.Trim() + ' ' + $silentArgs).Trim()

        Write-Host "Uninstall command: $uninstallPath"
        Write-Host "Uninstall args: $finalArgs"

        $processOptions = @{
            FilePath     = $uninstallPath
            ArgumentList = $finalArgs
            PassThru     = $true
            Wait         = $true
        }

        $process = Start-Process @processOptions
        $exitCode = $process.ExitCode

        Write-Host "Uninstall exit code: $exitCode"
        break
    }
}

if (-not $foundUninstaller) {
    Write-Host "Uninstaller for WinSCP not found."
    $exitCode = 1
}

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
}

Exit $exitCode
