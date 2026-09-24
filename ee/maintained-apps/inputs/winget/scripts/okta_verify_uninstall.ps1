# Okta Verify installs as a WiX Burn bundle. Prefer the bundle's registry
# uninstaller so all chained packages are removed, and fall back to the cached
# bootstrapper under Package Cache when the registry command is unavailable.

$productCode = "{008b801f-b8a1-40df-911b-a77c60e029c7}"
$displayNameLike = "Okta Verify*"
$publisherLike = "Okta*"
$expectedExitCodes = @(0, 1641, 3010)

function Split-UninstallCommand {
    param([string]$raw)

    if ($raw -match '^\s*"([^"]+)"\s*(.*)$') {
        return @($matches[1], $matches[2].Trim())
    }
    if ($raw -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
        return @($matches[1], $matches[2].Trim())
    }
    if ($raw -match '^\s*(\S+)\s*(.*)$') {
        return @($matches[1], $matches[2].Trim())
    }

    throw "Could not parse uninstall string: $raw"
}

function Invoke-Uninstaller {
    param([string]$exePath, [string]$existingArgs)

    if ($exePath -match '(?i)(^|\\)msiexec(\.exe)?$') {
        $existingArgs = ($existingArgs -replace '(?i)/i', '/x') -replace '(?i)/uninstall', ''
        if ($existingArgs -notmatch '(?i)/x') { $existingArgs = ("/x $existingArgs").Trim() }
        if ($existingArgs -notmatch '(?i)/q(n|uiet)?') { $existingArgs = ("$existingArgs /qn").Trim() }
        if ($existingArgs -notmatch '(?i)/norestart') { $existingArgs = ("$existingArgs /norestart").Trim() }
    } else {
        if ($existingArgs -notmatch '(?i)/uninstall') { $existingArgs = ("$existingArgs /uninstall").Trim() }
        if ($existingArgs -notmatch '(?i)/quiet') { $existingArgs = ("$existingArgs /quiet").Trim() }
        if ($existingArgs -notmatch '(?i)/norestart') { $existingArgs = ("$existingArgs /norestart").Trim() }
    }

    Write-Host "Uninstall command: $exePath"
    Write-Host "Uninstall args: $existingArgs"

    $process = Start-Process -FilePath $exePath -ArgumentList $existingArgs -NoNewWindow -PassThru -Wait
    return $process.ExitCode
}

$paths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$candidates = @()
foreach ($p in $paths) {
    foreach ($keyName in @($productCode, $productCode.Trim('{}'))) {
        $keyPath = "$p\$keyName"
        if (Test-Path $keyPath) {
            $entry = Get-ItemProperty $keyPath -ErrorAction SilentlyContinue
            if ($entry) { $candidates += $entry }
        }
    }

    $candidates += Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
        $_.DisplayName -like $displayNameLike -and $_.Publisher -like $publisherLike
    }
}

$entry = $candidates | Where-Object { $_.QuietUninstallString } | Select-Object -First 1
if (-not $entry) {
    $entry = $candidates | Where-Object {
        $_.UninstallString -and $_.UninstallString -notmatch '(?i)msiexec'
    } | Select-Object -First 1
}
if (-not $entry) {
    $entry = $candidates | Where-Object { $_.UninstallString } | Select-Object -First 1
}

$exitCode = $null

try {
    Stop-Process -Name "OktaVerify", "Okta Verify" -Force -ErrorAction SilentlyContinue

    if ($entry) {
        $raw = if ($entry.QuietUninstallString) { $entry.QuietUninstallString } else { $entry.UninstallString }
        $commandParts = Split-UninstallCommand -raw $raw
        $exitCode = Invoke-Uninstaller -exePath $commandParts[0] -existingArgs $commandParts[1]
    }

    if ($null -eq $exitCode) {
        foreach ($cacheKey in @($productCode, $productCode.Trim('{}'))) {
            $cached = Get-ChildItem -Path "C:\ProgramData\Package Cache\$cacheKey" -Filter *.exe -ErrorAction SilentlyContinue | Select-Object -First 1
            if ($cached) {
                $exitCode = Invoke-Uninstaller -exePath $cached.FullName -existingArgs ""
                break
            }
        }
    }

    if ($null -eq $exitCode) {
        Write-Host "Uninstall entry not found for $displayNameLike"
        Exit 0
    }

    Write-Host "Uninstall exit code: $exitCode"
    if ($expectedExitCodes -contains $exitCode) { Exit 0 }
    Exit $exitCode
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
