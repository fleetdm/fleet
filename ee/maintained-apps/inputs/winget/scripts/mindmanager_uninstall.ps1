# Uninstall MindManager.
#
# The installer registers a versioned DisplayName ("MindManager 25"), so match the
# product family. It is recommended to keep this specific enough to avoid removing
# unintended software.
$softwareNameLike = "MindManager *"

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
        #   "C:\path with spaces\setup.exe" /uninstall   (quoted)
        #   C:\path with spaces\setup.exe /uninstall      (unquoted, may contain spaces)
        #   MsiExec.exe /X{GUID} / /I{GUID}               (bare token, MSI product)
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

        $uninstallArgs = $existingArgs
        if ($exe -match '(?i)msiexec') {
            # MSI product: ensure /X for removal and silent flags.
            if ($uninstallArgs -match '(?i)/I(\{[^}]+\})') {
                $uninstallArgs = $uninstallArgs -replace '(?i)/I(\{[^}]+\})', '/X$1'
            }
            if ($uninstallArgs -notmatch '(?i)/qn|/quiet') {
                $uninstallArgs = "$uninstallArgs /qn".Trim()
            }
            if ($uninstallArgs -notmatch '(?i)/norestart') {
                $uninstallArgs = "$uninstallArgs /norestart".Trim()
            }
        } else {
            # Bootstrapper exe: pass silent switches through to the inner MSI.
            if ($uninstallArgs -notmatch '(?i)/S\b') {
                $uninstallArgs = "$uninstallArgs /S".Trim()
            }
            if ($uninstallArgs -notmatch '(?i)/v') {
                $uninstallArgs = "$uninstallArgs /v/qn".Trim()
            }
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
