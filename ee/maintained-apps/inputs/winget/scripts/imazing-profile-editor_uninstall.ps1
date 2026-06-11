# Uninstall iMazing Profile Editor (Inno Setup).
#
# It is recommended to use an exact software name here if possible to avoid
# uninstalling unintended software.
$softwareNameLike = "*iMazing Profile Editor*"

# Inno Setup uninstallers run silently with these flags.
$silentArgs = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$exitCode = 0

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path $paths `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath }

$foundUninstaller = $false
foreach ($key in $uninstallKeys) {
    if ($key.DisplayName -like $softwareNameLike) {
        $foundUninstaller = $true

        # Prefer QuietUninstallString when present.
        $uninstallCommand = if ($key.QuietUninstallString) {
            $key.QuietUninstallString
        } else {
            $key.UninstallString
        }

        # Parse the UninstallString defensively. It can be:
        #   "C:\path with spaces\unins000.exe" /SILENT   (quoted)
        #   C:\path with spaces\unins000.exe /SILENT      (unquoted, may contain spaces)
        #   MsiExec.exe /X{GUID}                          (bare token)
        $exe = $null
        $existingArgs = ""
        if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
            $exe = $matches[1]
            $existingArgs = $matches[2].Trim()
        } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
            $exe = $matches[1]
            $existingArgs = $matches[2].Trim()
        } elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
            $exe = $matches[1]
            $existingArgs = $matches[2].Trim()
        } else {
            Throw "Could not parse uninstall command: $uninstallCommand"
        }

        # Ensure silent flags are present (skip if QuietUninstallString already has them).
        $uninstallArgs = $existingArgs
        if ($existingArgs -notmatch '(?i)/VERYSILENT|/SILENT') {
            $uninstallArgs = "$existingArgs $silentArgs".Trim()
        }

        Write-Host "Uninstall command: $exe"
        Write-Host "Uninstall args: $uninstallArgs"

        $processOptions = @{
            FilePath = $exe
            PassThru = $true
            Wait = $true
        }
        if ($uninstallArgs -ne '') {
            $processOptions.ArgumentList = $uninstallArgs
        }

        $process = Start-Process @processOptions
        $exitCode = $process.ExitCode
        Write-Host "Uninstall exit code: $exitCode"
        break
    }
}

if (-not $foundUninstaller) {
    Write-Host "Uninstall entry not found for $softwareNameLike"
    Exit 0
}

Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
