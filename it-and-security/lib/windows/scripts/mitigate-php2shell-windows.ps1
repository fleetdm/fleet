# mitigate-php2shell-windows.ps1
# Inventory Windows PHP builds and apply PHP's interim PHP2Shell mitigation
# (CVE-2026-109618, CVE-2026-109619, disclosed 2026-10-21).
#   1. Finds PHP builds in the common install locations, in running php.exe, php-cgi.exe, and
#      php-win.exe processes, in IIS FastCGI handlers, and in Apache LoadModule lines, and reports
#      each build's file version against the fixed releases (8.5.12, 8.4.27, 8.3.36, 8.2.35).
#      It never runs a discovered php.exe: this script runs as SYSTEM and a local user can plant one.
#      A script can't replace a Windows build in place: download the fixed zip from
#      https://windows.php.net/download/ and swap the directory, or update XAMPP, WAMP, or Laragon.
#   2. Sets session.upload_progress.enabled = 0 and fatal_error_backtraces = 0 in every ini file
#      PHP can load for those builds, following PHP's search order: the IIS -c argument and PHPRC,
#      Apache's PHPIniDir, the machine PHPRC, HKLM\SOFTWARE\PHP IniFilePath, the build directory
#      (Apache's bin directory for mod_php), and the Windows directory, with php-cgi-fcgi.ini and
#      php-apache2handler.ini as well as php.ini. Creates php.ini where PHP looks first when it
#      would load none, keeps a .pre-php2shell backup of each file, and also sets both directives
#      in PHP_INI_SCAN_DIR files that override them, because those load after php.ini.
#   3. Restarts IIS (iisreset) and Apache services when a php-cgi.exe or httpd.exe process started
#      before the newest ini change, because PHP reads its ini only at startup.
# Partial mitigation only: the out-of-memory path (CVE-2026-109618) stays open until the
# build is replaced. Idempotent: safe to re-run. Built for Fleet "Run script" (runs as SYSTEM).
# Rollback: copy each .pre-php2shell backup back over its ini file and restart the same services.
# Exit: 0 = applied or nothing to do, 2 = something needs a look (read output), 1 = error.
# Advisory: https://github.com/php/php-src/security/advisories/GHSA-6wmf-4rv5-p3rp

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version 2
$script:ExitCode = 0

# Write-Host keeps log lines out of the values functions return.
function Log([string]$Message) { Write-Host $Message }
function Warn([string]$Message) { Write-Host "WARN: $Message"; $script:ExitCode = 2 }

function Get-FixedVersion([version]$v) {
    switch ("$($v.Major).$($v.Minor)") {
        '8.5' { return [version]'8.5.12' }
        '8.4' { return [version]'8.4.27' }
        '8.3' { return [version]'8.3.36' }
        '8.2' { return [version]'8.2.35' }
        default { return $null }
    }
}

# Apache and IIS settings can be quoted, use forward slashes, or be relative to a root.
function ConvertTo-WindowsPath([string]$Path, [string]$Root) {
    $p = $Path.Trim().Trim('"').Replace('/', '\')
    if ($Root -and -not [IO.Path]::IsPathRooted($p)) { $p = Join-Path $Root $p }
    try { $p = [IO.Path]::GetFullPath($p) } catch { return $p }
    if ($p.Length -gt 3) { $p = $p.TrimEnd('\') }
    return $p
}

# IIS FastCGI handlers that run PHP, with the ini locations they pass (-c, PHPRC, monitorChangesTo)
# and their PHP_INI_SCAN_DIR.
function Get-IisPhpApps {
    $config = Join-Path $env:windir 'System32\inetsrv\config\applicationHost.config'
    if (-not (Test-Path -LiteralPath $config)) { return }
    $xml = New-Object System.Xml.XmlDocument
    try { $xml.Load($config) } catch { Warn "cannot read $config, so IIS PHP handlers aren't covered: $_"; return }
    foreach ($app in @($xml.SelectNodes('/configuration/system.webServer/fastCgi/application'))) {
        $exe = [Environment]::ExpandEnvironmentVariables($app.GetAttribute('fullPath'))
        if ($exe -notmatch '\\php(-cgi)?\.exe$' -or -not (Test-Path -LiteralPath $exe)) { continue }
        $vars = @{}
        foreach ($var in @($app.SelectNodes('environmentVariables/environmentVariable'))) {
            $vars[$var.GetAttribute('name')] = [Environment]::ExpandEnvironmentVariables($var.GetAttribute('value'))
        }
        $ini = @()
        $c = [regex]::Match($app.GetAttribute('arguments'), '(?:^|\s)-c\s*(?:"([^"]+)"|(\S+))')
        if ($c.Success) { $ini += $c.Groups[1].Value + $c.Groups[2].Value }
        if ($vars['PHPRC']) { $ini += $vars['PHPRC'] }
        $monitor = [Environment]::ExpandEnvironmentVariables($app.GetAttribute('monitorChangesTo'))
        if ($monitor -like '*.ini') { $ini += $monitor }
        [pscustomobject]@{
            Dir      = ConvertTo-WindowsPath (Split-Path -Parent $exe)
            IniPaths = @($ini | ForEach-Object { ConvertTo-WindowsPath $_ })
            ScanDir  = $vars['PHP_INI_SCAN_DIR']
        }
    }
}

# Expands ${VAR} from Apache Define lines; a few passes cover definitions built from other definitions.
function Expand-ApacheVars([string]$Value, [hashtable]$Defines) {
    for ($i = 0; $i -lt 5 -and $Value.Contains('${'); $i++) {
        foreach ($k in @($Defines.Keys)) { $Value = $Value.Replace('${' + $k + '}', $Defines[$k]) }
    }
    return $Value
}

# Apache installs running as a service or process that load mod_php, with the module directories,
# their PHPIniDir, and the bin directory, which mod_php searches instead of the PHP directory.
function Get-ApachePhp {
    $exes = @(Get-CimInstance Win32_Service -ErrorAction SilentlyContinue | Where-Object { $_.PathName -match 'httpd\.exe' } |
        ForEach-Object { [regex]::Match($_.PathName, '^\s*"?([^"]*?httpd\.exe)').Groups[1].Value })
    $exes += @(Get-Process -Name httpd -ErrorAction SilentlyContinue | ForEach-Object { $_.Path })
    foreach ($exe in @($exes | Where-Object { $_ -and (Test-Path -LiteralPath $_) } | Sort-Object -Unique)) {
        $bin = Split-Path -Parent $exe
        $root = Split-Path -Parent $bin
        $lines = @(Get-ChildItem -LiteralPath (Join-Path $root 'conf') -Recurse -Filter *.conf -ErrorAction SilentlyContinue |
            ForEach-Object { [IO.File]::ReadAllLines($_.FullName) })
        $defines = @{}
        foreach ($l in $lines) { if ($l -match '^\s*Define\s+(\w+)\s+(.+)$') { $defines[$Matches[1]] = $Matches[2].Trim().Trim('"') } }
        $modules = @(); $iniDirs = @()
        foreach ($l in $lines) {
            if ($l -notmatch '^\s*(LoadModule\s+php\w*_module|PHPIniDir)\s+(.+)$') { continue }
            $directive = ($Matches[1] -split '\s+')[0]
            $value = Expand-ApacheVars $Matches[2] $defines
            if ($value.Contains('${')) { Warn "Apache in $root sets $directive to $value, which this script can't resolve; apply the mitigation there by hand"; continue }
            $path = ConvertTo-WindowsPath $value $root
            if ($directive -eq 'LoadModule') { $modules += Split-Path -Parent $path } else { $iniDirs += $path }
        }
        if ($modules.Count -gt 0) {
            [pscustomobject]@{ Root = $root; Bin = $bin; ModuleDirs = @($modules | Sort-Object -Unique); IniDirs = @($iniDirs | Sort-Object -Unique) }
        }
    }
}

# A 32-bit PHP on 64-bit Windows reads the WOW6432Node registry view, so check the build's PE machine type.
function Test-Php32Bit([string]$Dir) {
    foreach ($exe in 'php.exe', 'php-cgi.exe') {
        $path = Join-Path $Dir $exe
        if (-not (Test-Path -LiteralPath $path)) { continue }
        $fs = [IO.File]::OpenRead($path)
        try {
            $reader = New-Object IO.BinaryReader($fs)
            $fs.Position = 0x3C
            $fs.Position = $reader.ReadInt32() + 4
            return $reader.ReadUInt16() -eq 0x14C
        } finally { $fs.Dispose() }
    }
    return $false
}

# PHP opens the first existing key of HKLM\SOFTWARE\PHP\<x.y.z|x.y|x> and HKLM\SOFTWARE\PHP in its own
# registry view, and reads IniFilePath from that key only.
function Get-RegistryIniPath([version]$Version, [bool]$Is32Bit) {
    $root = if ($Is32Bit -and [Environment]::Is64BitOperatingSystem) { 'HKLM:\SOFTWARE\WOW6432Node\PHP' } else { 'HKLM:\SOFTWARE\PHP' }
    $keys = @()
    if ($Version) { $keys = @("$root\$($Version.Major).$($Version.Minor).$($Version.Build)", "$root\$($Version.Major).$($Version.Minor)", "$root\$($Version.Major)") }
    foreach ($key in $keys + $root) {
        if (-not (Test-Path -LiteralPath $key)) { continue }
        $p = Get-ItemProperty -LiteralPath $key -Name IniFilePath -ErrorAction SilentlyContinue
        if ($p -and $p.IniFilePath) { return ConvertTo-WindowsPath $p.IniFilePath }
        return
    }
}

# Directories that hold a PHP build: common layouts, running PHP processes, IIS FastCGI handlers, and Apache mod_php.
function Find-PhpDirs([object[]]$IisApps, [object[]]$Apaches) {
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
    foreach ($app in $IisApps) { $dirs.Add($app.Dir) }
    foreach ($apache in $Apaches) {
        foreach ($d in $apache.ModuleDirs) { if (Test-Path -LiteralPath $d -PathType Container) { $dirs.Add($d) } }
    }
    return @($dirs | ForEach-Object { ConvertTo-WindowsPath $_ } | Sort-Object -Unique)
}

function Get-PhpVersion([string]$Dir) {
    foreach ($exe in 'php.exe', 'php-cgi.exe') {
        $path = Join-Path $Dir $exe
        if (-not (Test-Path -LiteralPath $path)) { continue }
        $info = (Get-Item -LiteralPath $path).VersionInfo
        foreach ($candidate in @($info.ProductVersion, $info.FileVersion)) {
            $m = [regex]::Match([string]$candidate, '^\d+\.\d+\.\d+')
            if ($m.Success) { return [version]$m.Value }
        }
    }
    return $null
}

# Ini files PHP can load from search locations that are directories or ini files.
function Resolve-IniFiles([string[]]$Paths) {
    foreach ($p in @($Paths | Where-Object { $_ })) {
        if (Test-Path -LiteralPath $p -PathType Leaf) { $p; continue }
        if (-not (Test-Path -LiteralPath $p -PathType Container)) { continue }
        foreach ($name in 'php-cgi-fcgi.ini', 'php-apache2handler.ini', 'php.ini') {
            $f = Join-Path $p $name
            if (Test-Path -LiteralPath $f -PathType Leaf) { $f }
        }
    }
}

# PHP applies [HOST=...] and [PATH=...] sections (CGI and FastCGI) to one host or path, so the directive
# goes in global scope before the first such section, and definitions inside them are commented out.
function Set-IniDirective([System.Collections.Generic.List[string]]$Lines, [string]$Name, [string]$Value) {
    $pattern = '^\s*;?\s*' + [regex]::Escape($Name) + '\s*='
    $done = $false
    $scoped = $false
    $firstScoped = -1
    for ($i = 0; $i -lt $Lines.Count; $i++) {
        if ($Lines[$i] -match '^\s*\[') {
            $scoped = $Lines[$i] -match '^\s*\[\s*(HOST|PATH)'
            if ($scoped -and $firstScoped -lt 0) { $firstScoped = $i }
            continue
        }
        if ($Lines[$i] -notmatch $pattern) { continue }
        if (-not $scoped -and -not $done) { $Lines[$i] = "$Name = $Value"; $done = $true }
        elseif ($Lines[$i] -notmatch '^\s*;') { $Lines[$i] = '; ' + $Lines[$i] + '  ; disabled by mitigate-php2shell-windows.ps1' }
    }
    if (-not $done) {
        $block = @('; PHP2Shell interim mitigation (CVE-2026-109618, CVE-2026-109619). Not a substitute for updating PHP.', "$Name = $Value")
        if ($firstScoped -ge 0) {
            if ($firstScoped -gt 0 -and $Lines[$firstScoped - 1] -ne '') { $block = @('') + $block }
            $Lines.InsertRange($firstScoped, [string[]]($block + ''))
        } else {
            if ($Lines.Count -eq 0 -or $Lines[$Lines.Count - 1] -ne '') { $Lines.Add('') }
            $Lines.AddRange([string[]]$block)
        }
    }
}

function Set-PhpMitigation([string]$Path) {
    $exists = Test-Path -LiteralPath $Path -PathType Leaf
    $lines = New-Object System.Collections.Generic.List[string]
    if ($exists) { foreach ($l in [IO.File]::ReadAllLines($Path)) { $lines.Add($l) } }
    $before = $lines -join "`n"
    Set-IniDirective -Lines $lines -Name 'session.upload_progress.enabled' -Value '0'
    Set-IniDirective -Lines $lines -Name 'fatal_error_backtraces' -Value '0'
    $after = $lines -join "`n"
    if ($before -eq $after) { Log "INI: $Path already has the mitigation"; return }
    if ($exists) {
        $backup = "$Path.pre-php2shell"
        if (-not (Test-Path -LiteralPath $backup)) { Copy-Item -LiteralPath $Path -Destination $backup }
    }
    [IO.File]::WriteAllText($Path, $after + "`n", (New-Object System.Text.UTF8Encoding($false)))
    Log ("INI: {0} {1}" -f $Path, $(if ($exists) { 'updated' } else { 'created (PHP loaded no ini file)' }))
}

# PHP reads its ini only at startup, so php-cgi.exe and httpd.exe (mod_php) processes that started
# before the newest ini change still run with the old settings.
function Get-StaleHosts([datetime]$Since) {
    @(Get-Process -Name php-cgi, httpd -ErrorAction SilentlyContinue | Where-Object { try { $_.StartTime -lt $Since } catch { $true } })
}

function Restart-PhpHosts {
    $w3svc = Get-Service -Name W3SVC -ErrorAction SilentlyContinue
    if ($w3svc -and $w3svc.Status -eq 'Running') {
        & iisreset /restart | Out-Null
        if ($LASTEXITCODE -eq 0) { Log 'RESTARTED: IIS (iisreset), php-cgi.exe workers reload their ini' } else { Warn 'iisreset failed; recycle the application pools by hand' }
    }
    foreach ($svc in @(Get-CimInstance Win32_Service -ErrorAction SilentlyContinue | Where-Object { $_.State -eq 'Running' -and $_.PathName -match 'httpd\.exe|apache' })) {
        try { Restart-Service -Name $svc.Name -Force; Log "RESTARTED: $($svc.Name) (Apache)" } catch { Warn "failed to restart $($svc.Name): $_" }
    }
}

try {
    $iisApps = @(Get-IisPhpApps)
    $apaches = @(Get-ApachePhp)
    $dirs = @(Find-PhpDirs -IisApps $iisApps -Apaches $apaches)
    if ($dirs.Count -eq 0) {
        Log 'No PHP build found in the common locations, running processes, IIS FastCGI handlers, or Apache configuration.'
        $xampp = Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*', 'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*' -ErrorAction SilentlyContinue | Where-Object { $_.PSObject.Properties['DisplayName'] -and $_.DisplayName -like 'XAMPP*' }
        if ($xampp) { Warn "XAMPP $($xampp.DisplayVersion) is registered but C:\xampp\php\php.exe was not found; check its install path" }
        exit $script:ExitCode
    }
    $phprc = [Environment]::GetEnvironmentVariable('PHPRC', 'Machine')
    if ($phprc) { $phprc = ConvertTo-WindowsPath $phprc }

    Log '----- PHP builds -----'
    # One entry per way PHP runs, listing the ini locations it searches in PHP's order.
    $searches = New-Object System.Collections.Generic.List[object]
    foreach ($dir in $dirs) {
        $v = Get-PhpVersion $dir
        if (-not $v) { Warn "PHP: $dir has no readable version; replace it with a fixed build"; continue }
        $fixed = Get-FixedVersion $v
        if (-not $fixed) { Warn "PHP: $dir is $v, a branch with no fixed Windows build; move to 8.2 or newer" }
        elseif ($v -lt $fixed) { Warn "PHP: $dir is $v, UPDATE to $fixed from https://windows.php.net/download/" }
        else { Log "PHP: $dir is $v, FIXED" }
        $registry = @(Get-RegistryIniPath -Version $v -Is32Bit (Test-Php32Bit $dir))
        $iisPaths = @($iisApps | Where-Object { $_.Dir -eq $dir } | ForEach-Object { $_.IniPaths })
        $searches.Add(@($iisPaths + $phprc + $registry + $dir + $env:windir))
        foreach ($apache in @($apaches | Where-Object { $_.ModuleDirs -contains $dir })) {
            $searches.Add(@($apache.IniDirs + $phprc + $registry + $apache.Bin + $env:windir))
        }
    }

    Log '----- ini files -----'
    $files = New-Object System.Collections.Generic.List[string]
    foreach ($search in $searches) {
        $found = @(Resolve-IniFiles $search)
        if ($found.Count -eq 0) {
            $first = @($search | Where-Object { $_ -and (Test-Path -LiteralPath $_ -PathType Container) })[0]
            $found = @(Join-Path $first 'php.ini')
        }
        foreach ($f in $found) { $files.Add($f) }
    }
    $scanDirs = @(@([Environment]::GetEnvironmentVariable('PHP_INI_SCAN_DIR', 'Machine')) + @($iisApps | ForEach-Object { $_.ScanDir }) |
        Where-Object { $_ } | ForEach-Object { $_ -split ';' } | Where-Object { $_ -and (Test-Path -LiteralPath $_ -PathType Container) })
    foreach ($scanFile in @($scanDirs | ForEach-Object { Get-ChildItem -LiteralPath $_ -Filter *.ini -File -ErrorAction SilentlyContinue })) {
        if ([IO.File]::ReadAllText($scanFile.FullName) -match '(?m)^\s*(session\.upload_progress\.enabled|fatal_error_backtraces)\s*=') { $files.Add($scanFile.FullName) }
    }
    $files = @($files | ForEach-Object { ConvertTo-WindowsPath $_ } | Sort-Object -Unique)
    foreach ($f in $files) { Set-PhpMitigation $f }

    Log '----- PHP hosts -----'
    $newest = @($files | Where-Object { Test-Path -LiteralPath $_ } | ForEach-Object { (Get-Item -LiteralPath $_).LastWriteTime } | Sort-Object -Descending)[0]
    if (@(Get-StaleHosts $newest).Count -eq 0) {
        Log 'No php-cgi.exe or httpd.exe process started before the last ini change; nothing to restart.'
    } else {
        Restart-PhpHosts
        Start-Sleep -Seconds 3
        $left = @(Get-StaleHosts $newest)
        if ($left.Count -gt 0) { Warn ("still running with the old settings: {0}. Restart them by hand (XAMPP control panel, nginx, or another spawner)." -f (($left | ForEach-Object { "$($_.Name).exe pid $($_.Id)" }) -join ', ')) }
    }
    if ($script:ExitCode -eq 0) { Log 'DONE: every build is fixed and the mitigation is in place.' }
    else { Log 'DONE with warnings (exit 2): read the lines above. The mitigation is partial; replace the build.' }
    exit $script:ExitCode
} catch {
    Write-Host "ERROR: $_"
    exit 1
}
