# Uninstalls Raspberry Pi Imager (Inno Setup). Locates the ARP entry by exact
# DisplayName and runs its (Quiet)UninstallString with the Inno silent flags.
$softwareName = "Raspberry Pi Imager"

$uninstallArgs = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART"

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

$exitCode = 0

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path @($machineKey, $machineKey32on64) `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue }

$foundUninstaller = $false
foreach ($key in $uninstallKeys) {
    if ($key.DisplayName -eq $softwareName) {
        $foundUninstaller = $true
        $uninstallCommand = if ($key.QuietUninstallString) {
            $key.QuietUninstallString
        } else {
            $key.UninstallString
        }

        # Parse the command defensively: quoted path, unquoted path that may
        # contain spaces (captured through ".exe"), or a bare token.
        $trailingArgs = ""
        if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
            $uninstallCommand = $matches[1]
            $trailingArgs = $matches[2]
        } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
            $uninstallCommand = $matches[1]
            $trailingArgs = $matches[2]
        } elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
            $uninstallCommand = $matches[1]
            $trailingArgs = $matches[2]
        } else {
            Write-Host "Error: Could not parse uninstall string: $uninstallCommand"
            Exit 1
        }
        if ($trailingArgs -ne '') { $uninstallArgs = "$trailingArgs $uninstallArgs".Trim() }
        Write-Host "Uninstall command: $uninstallCommand"
        Write-Host "Uninstall args: $uninstallArgs"

        $processOptions = @{
            FilePath = $uninstallCommand
            PassThru = $true
            Wait = $true
        }
        if ($uninstallArgs -ne '') { $processOptions.ArgumentList = "$uninstallArgs" }

        $process = Start-Process @processOptions
        $exitCode = $process.ExitCode
        Write-Host "Uninstall exit code: $exitCode"
        break
    }
}

if (-not $foundUninstaller) {
    Write-Host "Uninstaller for '$softwareName' not found."
    $exitCode = 1
}

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
}

if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }
Exit $exitCode
