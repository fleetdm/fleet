# Uninstalls Foxit PDF Editor.
#
# The installer is a WiX bundle that registers two "Foxit PDF Editor" entries in
# Programs and Features: the bundle (an .exe UninstallString, takes /uninstall
# /quiet /norestart) and its chained MSI (MsiExec.exe /I{ProductCode}, needs
# msiexec /x /qn /norestart). The bundle is removed first since it takes the MSI
# with it; any MSI entry left behind is removed directly. Foxit's updater service
# and processes are stopped first so files aren't locked.

$softwareName = "Foxit PDF Editor"

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
$successCodes = @(0, 3010, 1641)

function Get-Entries {
    # Always yields a real (possibly empty) array so callers never iterate over $null.
    $keys = @(Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty -LiteralPath $_.PSPath -ErrorAction SilentlyContinue })
    return @($keys | Where-Object { $_ -and $_.DisplayName -eq $softwareName })
}

function Get-UninstallExe {
    param([string]$raw)
    if ($raw -match '^\s*"([^"]+)"') { return $matches[1] }
    if ($raw -match '(?i)^\s*(.+?\.exe)') { return $matches[1] }
    return $null
}

function Get-ProductCode {
    param($entry)
    if ($entry.PSChildName -match '^\{[0-9A-Fa-f-]+\}$') { return $entry.PSChildName }
    if ($entry.UninstallString -match '(\{[0-9A-Fa-f-]+\})') { return $matches[1] }
    return $null
}

$exitCode = 0

try {

foreach ($svc in @("FoxitPhantomPDFUpdateService", "FoxitPDFEditorUpdateService")) {
    Stop-Service -Name $svc -Force -ErrorAction SilentlyContinue
}
foreach ($p in @("FoxitPDFEditor", "FoxitPhantomPDF", "FoxitUpdater", "FoxitPDFEditorUpdateService")) {
    Stop-Process -Name $p -Force -ErrorAction SilentlyContinue
}
Start-Sleep -Seconds 3

$entries = @(Get-Entries)
if ($entries.Count -eq 0) {
    Write-Host "Uninstall entry not found for '$softwareName'."
    Exit 1
}

# Bundle first: its cached uninstaller removes the chained MSI as well.
foreach ($entry in $entries) {
    if ($entry.UninstallString -match '(?i)msiexec') { continue }
    $exe = Get-UninstallExe $entry.UninstallString
    if (-not $exe -or -not (Test-Path -LiteralPath $exe)) {
        Write-Host "Bundle uninstaller not found ($exe); removing the MSI directly."
        continue
    }
    Write-Host "Running bundle uninstall: $exe"
    $process = Start-Process -FilePath $exe -ArgumentList "/uninstall /quiet /norestart" -NoNewWindow -PassThru -Wait
    Write-Host "Bundle uninstall exit code: $($process.ExitCode)"
    if ($successCodes -notcontains $process.ExitCode) { $exitCode = $process.ExitCode }
}

# Any MSI entry still registered (older installs, or a bundle that could not run).
foreach ($entry in @(Get-Entries)) {
    if ($entry.UninstallString -notmatch '(?i)msiexec') { continue }
    $productCode = Get-ProductCode $entry
    if (-not $productCode) {
        Write-Host "Could not determine ProductCode for '$softwareName'."
        $exitCode = 1
        continue
    }
    Write-Host "Uninstalling product code: $productCode"
    $process = Start-Process msiexec.exe -ArgumentList "/x $productCode /qn /norestart" -NoNewWindow -PassThru -Wait
    Write-Host "Uninstall exit code: $($process.ExitCode)"
    # 1605 = product not installed; the entry was already stale.
    if ($successCodes -notcontains $process.ExitCode -and $process.ExitCode -ne 1605) { $exitCode = $process.ExitCode }
}

# A bundle whose cached uninstaller is gone leaves a dangling registration.
foreach ($entry in @(Get-Entries)) {
    if ($entry.UninstallString -match '(?i)msiexec') { continue }
    $exe = Get-UninstallExe $entry.UninstallString
    if ($exe -and (Test-Path -LiteralPath $exe)) { continue }
    if (-not $entry.PSPath) { continue }
    Write-Host "Removing orphaned registration: $($entry.PSChildName)"
    Remove-Item -LiteralPath $entry.PSPath -Recurse -Force -ErrorAction SilentlyContinue
}

} catch {
    Write-Host "Error: $_"
    Exit 1
}

if (@(Get-Entries).Count -gt 0) {
    Write-Host "'$softwareName' is still registered after uninstall."
    if ($exitCode -eq 0) { $exitCode = 1 }
}
Exit $exitCode
