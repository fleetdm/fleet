# Uninstall Jabra Direct (WiX burn bundle).
#
# It is recommended to use an exact software name here if possible to avoid
# uninstalling unintended software.
$softwareNameLike = "*Jabra Direct*"

# WiX burn bundles uninstall silently with these flags.
$silentArgs = "/uninstall /quiet /norestart"

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
        #   "C:\ProgramData\Package Cache\{GUID}\setup.exe" /uninstall   (quoted)
        #   C:\path with spaces\setup.exe /uninstall                     (unquoted, may contain spaces)
        #   MsiExec.exe /X{GUID}                                         (bare token)
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

        # Build silent uninstall args, avoiding duplicate switches.
        $uninstallArgs = $existingArgs
        if ($existingArgs -notmatch '(?i)/uninstall') {
            $uninstallArgs = "$existingArgs /uninstall".Trim()
        }
        if ($uninstallArgs -notmatch '(?i)/quiet|/silent') {
            $uninstallArgs = "$uninstallArgs /quiet".Trim()
        }
        if ($uninstallArgs -notmatch '(?i)/norestart') {
            $uninstallArgs = "$uninstallArgs /norestart".Trim()
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
