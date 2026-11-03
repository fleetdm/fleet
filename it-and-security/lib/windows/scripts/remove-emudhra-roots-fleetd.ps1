# Removes the emSign roots excluded by curl's CA bundle (eMudhra intermediate CA compromise,
# https://emudhra.we-have-a-problem.com) and the legacy Comodo AAA root from fleetd's osquery CA bundle,
# then reports whether Windows has Microsoft's blocklist for the compromised intermediates.
# Safe to run more than once.

param([string]$Bundle = "$env:ProgramFiles\Orbit\certs.pem")

$drop = @('emSign ECC Root CA - G3', 'emSign Root CA - C1', 'emSign ECC Root CA - C3', 'Comodo AAA Services root')

if (Test-Path -LiteralPath $Bundle) {
    $lines = [IO.File]::ReadAllLines($Bundle)
    $out = New-Object System.Collections.Generic.List[string]
    $skip = $false
    # certs.pem uses mk-ca-bundle's layout: a name line, an "=====" underline, then the PEM block.
    for ($i = 0; $i -lt $lines.Length; $i++) {
        if (-not $skip -and $drop -contains $lines[$i] -and $i + 1 -lt $lines.Length -and $lines[$i + 1] -match '^=+$') { $skip = $true }
        if (-not $skip) { $out.Add($lines[$i]) }
        if ($skip -and $lines[$i] -like '-----END CERTIFICATE-----*') { $skip = $false }
    }

    $before = @($lines | Where-Object { $_ -like '-----BEGIN CERTIFICATE-----*' }).Count
    $after = @($out | Where-Object { $_ -like '-----BEGIN CERTIFICATE-----*' }).Count
    $removed = $before - $after

    if ($removed -eq 0) {
        Write-Output "$Bundle doesn't contain the affected roots ($before certificates)."
    } elseif ($removed -gt 4 -or $after -lt 50) {
        # Guard against a malformed bundle stripping more than the four named roots.
        Write-Error "Unexpected result ($before -> $after certificates), leaving $Bundle unchanged."
        exit 1
    } else {
        $backup = "$Bundle.bak-emudhra"
        if (-not (Test-Path -LiteralPath $backup)) { Copy-Item -LiteralPath $Bundle -Destination $backup }
        # Write LF endings without a BOM to match the file fleetd installs.
        [IO.File]::WriteAllText($Bundle, (($out -join "`n") + "`n"), (New-Object System.Text.UTF8Encoding($false)))
        Write-Output "Removed $removed certificate(s) from $Bundle ($before -> $after). Backup: $backup"
    }
} else {
    Write-Output "No bundle at $Bundle, nothing to do."
}

$autoUpdate = Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\SystemCertificates\AuthRoot\AutoUpdate' -ErrorAction SilentlyContinue
if ($autoUpdate.DisallowedCertLastSyncTime) {
    $synced = [DateTime]::FromFileTimeUtc([BitConverter]::ToInt64($autoUpdate.DisallowedCertLastSyncTime, 0))
    Write-Output ("Disallowed certificate list last synced {0:u}" -f $synced)
    if ($synced -lt [DateTime]::new(2026, 10, 21, 0, 0, 0, [DateTimeKind]::Utc)) {
        Write-Output "That sync predates Microsoft's 2026-10-20 block of the eMudhra intermediates."
    }
} else {
    Write-Output "Windows hasn't synced the disallowed certificate list."
}
if ((Get-ItemProperty 'HKLM:\SOFTWARE\Policies\Microsoft\SystemCertificates\AuthRoot' -ErrorAction SilentlyContinue).DisableRootAutoUpdate -eq 1) {
    Write-Output "Automatic root certificate updates are turned off by policy. Deploy windows-distrust-emudhra.xml."
}
