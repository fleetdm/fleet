# mitigate-php2shell-windows.ps1
# Inventory Windows PHP builds and apply PHP's interim PHP2Shell mitigation
# (CVE-2026-109618, CVE-2026-109619, disclosed 2026-10-21).
#   1. Finds php.exe in the common install locations, in running php.exe, php-cgi.exe, and
#      php-win.exe processes, and in the IIS FastCGI configuration. Reports each build against
#      the fixed releases (8.5.12, 8.4.27, 8.3.36, 8.2.35). A script can't replace a Windows
#      build in place: download the fixed zip from https://windows.php.net/download/ and swap
#      the directory, or update XAMPP, WAMP, or Laragon.
#   2. Sets session.upload_progress.enabled = 0 and fatal_error_backtraces = 0 in each build's
#      loaded php.ini (creating one next to php.exe when none is loaded), keeping a
#      php.ini.pre-php2shell backup.
#   3. Restarts IIS (iisreset) and Apache services so php-cgi.exe workers reload php.ini.
# Partial mitigation only: the out-of-memory path (CVE-2026-109618) stays open until the
# build is replaced. Idempotent: safe to re-run. Built for Fleet "Run script" (runs as SYSTEM).
# Rollback: copy php.ini.pre-php2shell back over php.ini and restart the same services.
# Exit: 0 = applied or nothing to do, 2 = something needs a look (read output), 1 = error.
# Advisory: https://github.com/php/php-src/security/advisories/GHSA-6wmf-4rv5-p3rp

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version 2
$script:ExitCode = 0

function Log([string]$Message) { Write-Output $Message }
function Warn([string]$Message) { Write-Output "WARN: $Message"; if ($script:ExitCode -eq 0) { $script:ExitCode = 2 } }

function Get-FixedVersion([version]$v) {
    switch ("$($v.Major).$($v.Minor)") {
        '8.5' { return [version]'8.5.12' }
        '8.4' { return [version]'8.4.27' }
        '8.3' { return [version]'8.3.36' }
        '8.2' { return [version]'8.2.35' }
        default { return $null }
    }
}

# Directories that hold a php.exe: common layouts, running PHP processes, and IIS FastCGI handlers.
function Find-PhpDirs {
    $dirs = New-Object System.Collections.Generic.List[string]
    $patterns = @(
        'C:\php*', "$env:ProgramFiles\PHP", "$env:ProgramFiles\PHP\*", "${env:ProgramFiles(x86)}\PHP\*",
        'C:\xampp\php', 'C:\wamp64\bin\php\*', 'C:\wamp\bin\php\*', 'C:\laragon\bin\php\*',
        'C:\tools\php*', 'C:\inetpub\PHP\*', "$env:ProgramFiles\iis express\PHP\*"
    )
    foreach ($p in $patterns) {
        foreach ($d in @(Get-Item -Path $p -ErrorAction SilentlyContinue | Where-Object { $_.PSIsContainer })) {
            if (Test-Path -LiteralPath (Join-Path $d.FullName 'php.exe')) { $dirs.Add($d.FullName) }
        }
    }
    foreach ($proc in @(Get-Process -Name php, php-cgi, php-win -ErrorAction SilentlyContinue)) {
        if ($proc.Path) { $dirs.Add((Split-Path -Parent $proc.Path)) }
    }
    $appcmd = Join-Path $env:windir 'System32\inetsrv\appcmd.exe'
    if (Test-Path -LiteralPath $appcmd) {
        $xml = & $appcmd list config /section:system.webServer/fastCgi 2>$null | Out-String
        foreach ($m in [regex]::Matches($xml, 'fullPath="([^"]+php(?:-cgi)?\.exe)"')) {
            $exe = [Environment]::ExpandEnvironmentVariables($m.Groups[1].Value)
            if (Test-Path -LiteralPath $exe) { $dirs.Add((Split-Path -Parent $exe)) }
        }
    }
    return @($dirs | ForEach-Object { $_.TrimEnd('\') } | Sort-Object -Unique)
}

function Get-PhpVersion([string]$Dir) {
    $info = (Get-Item -LiteralPath (Join-Path $Dir 'php.exe')).VersionInfo
    foreach ($candidate in @($info.ProductVersion, $info.FileVersion)) {
        $m = [regex]::Match([string]$candidate, '^\d+\.\d+\.\d+')
        if ($m.Success) { return [version]$m.Value }
    }
    return $null
}

# The php.ini this build loads, or the path it would load from the php.exe directory.
function Get-PhpIniPath([string]$Dir) {
    $php = Join-Path $Dir 'php.exe'
    try {
        $loaded = (& $php -r 'echo php_ini_loaded_file();' 2>$null | Out-String).Trim()
        if ($loaded -and (Test-Path -LiteralPath $loaded)) { return @{ Path = $loaded; Exists = $true } }
    } catch { }
    return @{ Path = (Join-Path $Dir 'php.ini'); Exists = $false }
}

function Set-IniDirective([System.Collections.Generic.List[string]]$Lines, [string]$Name, [string]$Value) {
    $pattern = '^\s*;?\s*' + [regex]::Escape($Name) + '\s*='
    $done = $false
    for ($i = 0; $i -lt $Lines.Count; $i++) {
        if ($Lines[$i] -match $pattern) {
            if (-not $done) { $Lines[$i] = "$Name = $Value"; $done = $true }
            elseif ($Lines[$i] -notmatch '^\s*;') { $Lines[$i] = '; ' + $Lines[$i] + '  ; disabled by mitigate-php2shell-windows.ps1' }
        }
    }
    if (-not $done) {
        if ($Lines.Count -eq 0 -or $Lines[$Lines.Count - 1] -ne '') { $Lines.Add('') }
        $Lines.Add('; PHP2Shell interim mitigation (CVE-2026-109618, CVE-2026-109619). Not a substitute for updating PHP.')
        $Lines.Add("$Name = $Value")
    }
}

function Apply-Mitigation([string]$Dir) {
    $ini = Get-PhpIniPath $Dir
    $lines = New-Object System.Collections.Generic.List[string]
    if ($ini.Exists) { foreach ($l in [IO.File]::ReadAllLines($ini.Path)) { $lines.Add($l) } }
    $before = $lines -join "`n"
    Set-IniDirective $lines 'session.upload_progress.enabled' '0'
    Set-IniDirective $lines 'fatal_error_backtraces' '0'
    $after = $lines -join "`n"
    if ($before -eq $after) { Log "INI: $($ini.Path) already has the mitigation"; return $false }
    if ($ini.Exists) {
        $backup = "$($ini.Path).pre-php2shell"
        if (-not (Test-Path -LiteralPath $backup)) { Copy-Item -LiteralPath $ini.Path -Destination $backup }
    }
    [IO.File]::WriteAllText($ini.Path, $after + "`n", (New-Object System.Text.UTF8Encoding($false)))
    Log ("INI: {0} {1}" -f $ini.Path, $(if ($ini.Exists) { 'updated' } else { 'created (no php.ini was loaded)' }))
    return $true
}

function Restart-PhpHosts {
    $w3svc = Get-Service -Name W3SVC -ErrorAction SilentlyContinue
    if ($w3svc -and $w3svc.Status -eq 'Running') {
        & iisreset /restart | Out-Null
        if ($LASTEXITCODE -eq 0) { Log 'RESTARTED: IIS (iisreset), php-cgi.exe workers reload php.ini' } else { Warn 'iisreset failed; recycle the application pools by hand' }
    }
    foreach ($svc in @(Get-CimInstance Win32_Service -ErrorAction SilentlyContinue | Where-Object { $_.State -eq 'Running' -and $_.PathName -match 'httpd\.exe|apache' })) {
        try { Restart-Service -Name $svc.Name -Force; Log "RESTARTED: $($svc.Name) (Apache)" } catch { Warn "failed to restart $($svc.Name): $_" }
    }
    $stray = @(Get-Process -Name php-cgi -ErrorAction SilentlyContinue)
    if ($stray.Count -gt 0) { Warn "$($stray.Count) php-cgi.exe process(es) still running (nginx or a manual spawner?); restart them so they read php.ini" }
}

try {
    $dirs = Find-PhpDirs
    if ($dirs.Count -eq 0) {
        Log 'No PHP build found in the common locations, running processes, or IIS FastCGI configuration.'
        $xampp = Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*', 'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*' -ErrorAction SilentlyContinue | Where-Object { $_.PSObject.Properties['DisplayName'] -and $_.DisplayName -like 'XAMPP*' }
        if ($xampp) { Warn "XAMPP $($xampp.DisplayVersion) is registered but C:\xampp\php\php.exe was not found; check its install path" }
        exit 0
    }
    Log '----- PHP builds -----'
    $changed = $false
    foreach ($dir in $dirs) {
        $v = Get-PhpVersion $dir
        if (-not $v) { Warn "PHP: $dir has no readable version; replace it with a fixed build"; continue }
        $fixed = Get-FixedVersion $v
        if (-not $fixed) { Warn "PHP: $dir is $v, a branch with no fixed Windows build; move to 8.2 or newer" }
        elseif ($v -lt $fixed) { Warn "PHP: $dir is $v, UPDATE to $fixed from https://windows.php.net/download/" }
        else { Log "PHP: $dir is $v, FIXED" }
        if (Apply-Mitigation $dir) { $changed = $true }
    }
    Log '----- PHP hosts -----'
    if ($changed) { Restart-PhpHosts } else { Log 'No php.ini changed; nothing to restart.' }
    if ($script:ExitCode -eq 0) { Log 'DONE: every build is fixed and the mitigation is in place.' }
    else { Log 'DONE with warnings (exit 2): read the lines above. The mitigation is partial; replace the build.' }
    exit $script:ExitCode
} catch {
    Write-Output "ERROR: $_"
    exit 1
}
