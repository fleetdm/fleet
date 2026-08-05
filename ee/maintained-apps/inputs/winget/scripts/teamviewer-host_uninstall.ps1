# TeamViewer Host registers itself in Add/Remove Programs with the DisplayName
# "TeamViewer Host" (the MSI ProductName; there is no ARPDISPLAYNAME override).
#
# The trailing wildcard tolerates a version suffix if TeamViewer ever adds one,
# and the "Host" in the pattern keeps this from matching the full TeamViewer
# client, which registers as plain "TeamViewer" and ships as its own
# Fleet-maintained app (teamviewer/windows).
$softwareNameLike = "TeamViewer Host*"

# NSIS installers require /S flag for silent uninstall
$uninstallArgs = "/S"

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
        # Get the uninstall command. Some uninstallers do not include
        # 'QuietUninstallString' and require a flag to run silently.
        $uninstallCommand = if ($key.QuietUninstallString) {
            $key.QuietUninstallString
        } else {
            $key.UninstallString
        }

        # UninstallString comes in three shapes. TeamViewer's is unquoted and
        # contains a space ("C:\Program Files\TeamViewer\uninstall.exe"), so
        # capture through the .exe rather than splitting on the first space.
        $existingArgs = ''
        if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
            # Quoted path, optionally followed by args
            $uninstallCommand = $Matches[1]
            $existingArgs = $Matches[2].Trim()
        } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
            # Unquoted path that may contain spaces
            $uninstallCommand = $Matches[1]
            $existingArgs = $Matches[2].Trim()
        } elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
            # Bare token (e.g. MsiExec.exe /X{GUID})
            $uninstallCommand = $Matches[1]
            $existingArgs = $Matches[2].Trim()
        }

        # Append /S unless the registered command already runs silently
        if ($existingArgs -notmatch '(?i)(^|\s)/S(\s|$)') {
            $uninstallArgs = "$existingArgs /S".Trim()
        } else {
            $uninstallArgs = $existingArgs
        }

        Write-Host "Uninstall command: $uninstallCommand"
        Write-Host "Uninstall args: $uninstallArgs"

        $processOptions = @{
            FilePath = $uninstallCommand
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
