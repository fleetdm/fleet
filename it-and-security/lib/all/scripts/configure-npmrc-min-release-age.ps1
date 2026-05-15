# Enforce min-release-age >= 0.5 in per-user .npmrc on Windows.
# - Creates %USERPROFILE%\.npmrc with min-release-age=0.5 when missing.
# - Replaces assignments strictly below 0.5 with min-release-age=0.5.
# - Leaves assignments at or above 0.5 and all other lines unchanged.

$ErrorActionPreference = "Stop"
$minVal = 0.5

function Update-NpmrcFile {
    param([string]$Path)

    if (-not (Test-Path -LiteralPath $Path)) {
        Set-Content -LiteralPath $Path -Value "min-release-age=$minVal" -Encoding utf8
        return
    }

    $lines = Get-Content -LiteralPath $Path -Encoding utf8
    $out = New-Object System.Collections.Generic.List[string]
    $keptGood = $false

    foreach ($line in $lines) {
        if ($line -match '^\s*([^=#\s]+)\s*=\s*(.*)$') {
            $key = $Matches[1].ToLowerInvariant()
            if ($key -eq "min-release-age") {
                $raw = $Matches[2]
                $numPart = ($raw -split "#", 2)[0].Trim()
                $parsed = 0.0
                $ok = [double]::TryParse($numPart, [ref]$parsed)
                if (-not $ok) {
                    $out.Add("min-release-age=$minVal")
                    $keptGood = $true
                    continue
                }
                if ($parsed -ge $minVal) {
                    $out.Add($line)
                    $keptGood = $true
                }
                else {
                    $out.Add("min-release-age=$minVal")
                    $keptGood = $true
                }
                continue
            }
        }
        $out.Add($line)
    }

    if (-not $keptGood) {
        $out.Add("min-release-age=$minVal")
    }

    Set-Content -LiteralPath $Path -Value ($out -join "`n") -Encoding utf8
}

Get-ChildItem -Path "C:\Users" -Directory -ErrorAction SilentlyContinue | ForEach-Object {
    $name = $_.Name
    if ($name -in @("Public", "Default", "Default User", "All Users")) {
        return
    }
    $npmrc = Join-Path $_.FullName ".npmrc"
    Update-NpmrcFile -Path $npmrc
}

exit 0
