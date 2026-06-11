#Requires -RunAsAdministrator
<#
.SYNOPSIS
    CIS Windows 10 Enterprise Benchmark v3.0.0 - Account and Password Policy Remediation

.DESCRIPTION
    Configures account lockout and password policies to meet CIS Windows 10 Enterprise Benchmark v3.0.0
    requirements using secedit. This script is idempotent - safe to run multiple times.

.NOTES
    CIS Benchmark Controls Covered:
    - 1.1.1  Enforce password history: 24 or more passwords
    - 1.1.2  Maximum password age: 365 days or fewer (not 0)
    - 1.1.3  Minimum password age: 1 or more days
    - 1.1.4  Minimum password length: 14 or more characters
    - 1.1.5  Password must meet complexity requirements: Enabled
    - 1.1.6  Store passwords using reversible encryption: Disabled
    - 1.2.1  Account lockout duration: 15 or more minutes
    - 1.2.2  Account lockout threshold: 5 or fewer invalid attempts (not 0)
    - 1.2.3  Reset account lockout counter after: 15 or more minutes

    Requires: Administrator privileges
    Run as: SYSTEM or local Administrator
#>

[CmdletBinding(SupportsShouldProcess)]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Write-Status {
    param([string]$Message, [string]$Level = 'INFO')
    $timestamp = Get-Date -Format 'yyyy-MM-dd HH:mm:ss'
    Write-Output "[$timestamp] [$Level] $Message"
}

Write-Status "Starting CIS account and password policy remediation"

# Export current security policy to a temp file
$cfgFile  = Join-Path $env:TEMP "secpol_cis.cfg"
$sdbFile  = Join-Path $env:TEMP "secedit_cis.sdb"
$logFile  = Join-Path $env:TEMP "secedit_cis.log"

try {
    Write-Status "Exporting current security policy..."
    $exportResult = secedit /export /cfg $cfgFile /quiet
    if ($LASTEXITCODE -ne 0) {
        throw "secedit /export failed with exit code $LASTEXITCODE"
    }

    # Read the exported policy
    $content = Get-Content $cfgFile -Raw

    # --- Password Policy Settings ---

    # CIS 1.1.1: Enforce password history = 24 (or more)
    $content = $content -replace 'PasswordHistorySize\s*=\s*\d+', 'PasswordHistorySize = 24'
    if ($content -notmatch 'PasswordHistorySize') {
        $content = $content -replace '(\[System Access\])', "`$1`r`nPasswordHistorySize = 24"
    }

    # CIS 1.1.2: Maximum password age = 365 days (must not be 0; org may use shorter)
    $content = $content -replace 'MaximumPasswordAge\s*=\s*\d+', 'MaximumPasswordAge = 365'
    if ($content -notmatch 'MaximumPasswordAge') {
        $content = $content -replace '(\[System Access\])', "`$1`r`nMaximumPasswordAge = 365"
    }

    # CIS 1.1.3: Minimum password age = 1 day
    $content = $content -replace 'MinimumPasswordAge\s*=\s*\d+', 'MinimumPasswordAge = 1'
    if ($content -notmatch 'MinimumPasswordAge') {
        $content = $content -replace '(\[System Access\])', "`$1`r`nMinimumPasswordAge = 1"
    }

    # CIS 1.1.4: Minimum password length = 14 characters
    $content = $content -replace 'MinimumPasswordLength\s*=\s*\d+', 'MinimumPasswordLength = 14'
    if ($content -notmatch 'MinimumPasswordLength') {
        $content = $content -replace '(\[System Access\])', "`$1`r`nMinimumPasswordLength = 14"
    }

    # CIS 1.1.5: Password complexity = Enabled (1)
    $content = $content -replace 'PasswordComplexity\s*=\s*\d+', 'PasswordComplexity = 1'
    if ($content -notmatch 'PasswordComplexity') {
        $content = $content -replace '(\[System Access\])', "`$1`r`nPasswordComplexity = 1"
    }

    # CIS 1.1.6: Store password using reversible encryption = Disabled (0)
    $content = $content -replace 'ClearTextPassword\s*=\s*\d+', 'ClearTextPassword = 0'
    if ($content -notmatch 'ClearTextPassword') {
        $content = $content -replace '(\[System Access\])', "`$1`r`nClearTextPassword = 0"
    }

    # --- Account Lockout Settings ---

    # CIS 1.2.1: Account lockout duration = 15 minutes (0 = until admin unlocks; use 15+)
    $content = $content -replace 'LockoutDuration\s*=\s*-?\d+', 'LockoutDuration = 15'
    if ($content -notmatch 'LockoutDuration') {
        $content = $content -replace '(\[System Access\])', "`$1`r`nLockoutDuration = 15"
    }

    # CIS 1.2.2: Account lockout threshold = 5 (invalid logon attempts)
    $content = $content -replace 'LockoutBadCount\s*=\s*\d+', 'LockoutBadCount = 5'
    if ($content -notmatch 'LockoutBadCount') {
        $content = $content -replace '(\[System Access\])', "`$1`r`nLockoutBadCount = 5"
    }

    # CIS 1.2.3: Reset lockout counter after = 15 minutes
    $content = $content -replace 'ResetLockoutCount\s*=\s*\d+', 'ResetLockoutCount = 15'
    if ($content -notmatch 'ResetLockoutCount') {
        $content = $content -replace '(\[System Access\])', "`$1`r`nResetLockoutCount = 15"
    }

    # Write the modified policy back to disk
    Write-Status "Writing updated policy configuration..."
    Set-Content -Path $cfgFile -Value $content -Encoding Unicode

    # Apply the updated policy using secedit
    Write-Status "Applying updated security policy..."
    $applyResult = secedit /configure /db $sdbFile /cfg $cfgFile /areas SECURITYPOLICY /log $logFile /quiet
    if ($LASTEXITCODE -ne 0) {
        Write-Status "secedit /configure completed with exit code $LASTEXITCODE. Check $logFile for details." 'WARN'
    } else {
        Write-Status "Security policy applied successfully."
    }

} catch {
    Write-Status "ERROR: $_" 'ERROR'
    exit 1
} finally {
    # Clean up temp files
    @($cfgFile, $sdbFile, $logFile) | ForEach-Object {
        if (Test-Path $_) { Remove-Item $_ -Force -ErrorAction SilentlyContinue }
    }
}

Write-Status "Account and password policy remediation complete."
