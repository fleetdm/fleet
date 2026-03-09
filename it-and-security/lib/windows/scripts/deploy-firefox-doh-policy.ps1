# Deploy Firefox DNS over HTTPS (DoH) policy via policies.json
# This script creates/overwrites the Firefox enterprise policies.json file
# to enforce DNS over HTTPS with fallback enabled.
# Idempotent: safe to run multiple times.

$policiesContent = @'
{
  "policies": {
    "DNSOverHTTPS": {
      "Enabled": true,
      "Fallback": true,
      "Locked": true
    }
  }
}
'@

$firefoxPaths = @(
    "$env:ProgramFiles\Mozilla Firefox",
    "${env:ProgramFiles(x86)}\Mozilla Firefox"
)

$deployed = $false

foreach ($firefoxPath in $firefoxPaths) {
    if (Test-Path -Path $firefoxPath) {
        $distributionPath = Join-Path -Path $firefoxPath -ChildPath "distribution"
        if (-not (Test-Path -Path $distributionPath)) {
            New-Item -Path $distributionPath -ItemType Directory -Force | Out-Null
        }
        $policiesFile = Join-Path -Path $distributionPath -ChildPath "policies.json"
        Set-Content -Path $policiesFile -Value $policiesContent -Encoding UTF8 -Force
        Write-Output "Deployed policies.json to $policiesFile"
        $deployed = $true
    }
}

if (-not $deployed) {
    Write-Output "No Firefox installation found. Skipping policies.json deployment."
}
