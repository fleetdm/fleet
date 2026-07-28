# The registry DisplayName carries a version and build ("Rtools 4.5 (6768-6492)"),
# so match on a prefix. Require the publisher too, mirroring the manifest's exists
# query. Verified as "The R Foundation" against the setup stub's PE version
# resource (CompanyName, which Inno derives from AppPublisher). Requiring it here
# puts the value under test -- the validator's appExists looks up by name only, so
# a wrong exists-query publisher would otherwise ship undetected (cf. the Spyder
# finding in #50016).
$softwareName = "Rtools"
$softwareNameLike = "$softwareName*"
$softwarePublisher = "The R Foundation"
$uninstallArgs = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART"

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
$exitCode = 0

try {
    [array]$uninstallKeys = Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue }

    $foundUninstaller = $false
    foreach ($key in $uninstallKeys) {
        if ($key.DisplayName -like $softwareNameLike -and $key.Publisher -eq $softwarePublisher) {
            $foundUninstaller = $true
            $uninstallCommand = if ($key.QuietUninstallString) { $key.QuietUninstallString } else { $key.UninstallString }
            if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
                $uninstallCommand = $Matches[1]; if ($Matches[2]) { $uninstallArgs = "$($Matches[2]) $uninstallArgs".Trim() }
            } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
                $uninstallCommand = $Matches[1]; if ($Matches[2]) { $uninstallArgs = "$($Matches[2]) $uninstallArgs".Trim() }
            }
            Write-Host "Uninstall command: $uninstallCommand"; Write-Host "Uninstall args: $uninstallArgs"
            $processOptions = @{ FilePath = $uninstallCommand; PassThru = $true; Wait = $true }
            if ($uninstallArgs -ne '') { $processOptions.ArgumentList = $uninstallArgs }
            $process = Start-Process @processOptions
            $exitCode = $process.ExitCode; Write-Host "Uninstall exit code: $exitCode"; break
        }
    }
    # Nothing to remove is not a failure: uninstall scripts are idempotent here, as
    # in nordpass_uninstall.ps1 and windsurf_uninstall.ps1.
    if (-not $foundUninstaller) { Write-Host "Uninstall entry not found for '$softwareName'."; Exit 0 }
} catch { Write-Host "Error: $_"; Exit 1 }

Exit $exitCode
