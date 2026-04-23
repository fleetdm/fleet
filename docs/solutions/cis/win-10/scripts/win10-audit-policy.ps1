#Requires -RunAsAdministrator
<#
.SYNOPSIS
    CIS Windows 10 Enterprise Benchmark v3.0.0 - Audit Policy Remediation

.DESCRIPTION
    Configures Advanced Audit Policy settings to meet CIS Windows 10 Enterprise Benchmark v3.0.0
    requirements using auditpol.exe. This script is idempotent - safe to run
    multiple times.

.NOTES
    CIS Benchmark Controls Covered:
    - 17.1.1  Account Logon:    Audit Credential Validation          -> Success and Failure
    - 17.2.1  Account Mgmt:     Audit Application Group Management   -> Success and Failure
    - 17.2.4  Account Mgmt:     Audit Security Group Management      -> Success
    - 17.2.6  Account Mgmt:     Audit User Account Management        -> Success and Failure
    - 17.3.1  Detailed Tracking: Audit PNP Activity                  -> Success
    - 17.3.2  Detailed Tracking: Audit Process Creation              -> Success
    - 17.5.1  Logon/Logoff:     Audit Account Lockout               -> Failure
    - 17.5.2  Logon/Logoff:     Audit Group Membership              -> Success
    - 17.5.3  Logon/Logoff:     Audit Logoff                        -> Success
    - 17.5.4  Logon/Logoff:     Audit Logon                         -> Success and Failure
    - 17.5.5  Logon/Logoff:     Audit Other Logon/Logoff Events     -> Success and Failure
    - 17.5.6  Logon/Logoff:     Audit Special Logon                 -> Success
    - 17.6.1  Object Access:    Audit Detailed File Share            -> Failure
    - 17.6.2  Object Access:    Audit File Share                     -> Success and Failure
    - 17.6.3  Object Access:    Audit Other Object Access Events     -> Success and Failure
    - 17.6.4  Object Access:    Audit Removable Storage              -> Success and Failure
    - 17.7.1  Policy Change:    Audit Audit Policy Change            -> Success
    - 17.7.2  Policy Change:    Audit Authentication Policy Change   -> Success
    - 17.7.3  Policy Change:    Audit Authorization Policy Change    -> Success
    - 17.7.4  Policy Change:    Audit MPSSVC Rule-Level Policy Change -> Success and Failure
    - 17.7.5  Policy Change:    Audit Other Policy Change Events     -> Failure
    - 17.8.1  Privilege Use:    Audit Sensitive Privilege Use        -> Success and Failure
    - 17.9.1  System:           Audit IPsec Driver                   -> Success and Failure
    - 17.9.2  System:           Audit Other System Events            -> Success and Failure
    - 17.9.3  System:           Audit Security State Change          -> Success
    - 17.9.4  System:           Audit Security System Extension      -> Success
    - 17.9.5  System:           Audit System Integrity               -> Success and Failure

    Requires: Administrator privileges
    Note: Uses auditpol.exe which is available on all supported Windows versions.
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

function Set-AuditPolicy {
    param(
        [string]$Subcategory,
        [bool]$Success,
        [bool]$Failure
    )

    $successArg = if ($Success) { '/success:enable' } else { '/success:disable' }
    $failureArg = if ($Failure) { '/failure:enable' } else { '/failure:disable' }

    $result = auditpol /set /subcategory:"$Subcategory" $successArg $failureArg 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Status "Failed to set '$Subcategory': $result" 'WARN'
        return $false
    }
    $label = @()
    if ($Success) { $label += 'Success' }
    if ($Failure) { $label += 'Failure' }
    Write-Status "  Set '$Subcategory' = $($label -join ' and ')"
    return $true
}

Write-Status "Starting CIS audit policy remediation"

# Verify auditpol is available
if (-not (Get-Command auditpol -ErrorAction SilentlyContinue)) {
    Write-Status "auditpol.exe not found. Cannot configure audit policies." 'ERROR'
    exit 1
}

$errors = 0

# Account Logon
Write-Status "Configuring Account Logon audit policies..."
if (-not (Set-AuditPolicy -Subcategory 'Credential Validation' -Success $true -Failure $true)) { $errors++ }

# Account Management
Write-Status "Configuring Account Management audit policies..."
if (-not (Set-AuditPolicy -Subcategory 'Application Group Management' -Success $true -Failure $true)) { $errors++ }
if (-not (Set-AuditPolicy -Subcategory 'Security Group Management'    -Success $true -Failure $false)) { $errors++ }
if (-not (Set-AuditPolicy -Subcategory 'User Account Management'      -Success $true -Failure $true)) { $errors++ }

# Detailed Tracking
Write-Status "Configuring Detailed Tracking audit policies..."
if (-not (Set-AuditPolicy -Subcategory 'Plug and Play Events' -Success $true -Failure $false)) { $errors++ }
if (-not (Set-AuditPolicy -Subcategory 'Process Creation'     -Success $true -Failure $false)) { $errors++ }

# Logon/Logoff
Write-Status "Configuring Logon/Logoff audit policies..."
if (-not (Set-AuditPolicy -Subcategory 'Account Lockout'              -Success $false -Failure $true)) { $errors++ }
if (-not (Set-AuditPolicy -Subcategory 'Group Membership'             -Success $true  -Failure $false)) { $errors++ }
if (-not (Set-AuditPolicy -Subcategory 'Logoff'                       -Success $true  -Failure $false)) { $errors++ }
if (-not (Set-AuditPolicy -Subcategory 'Logon'                        -Success $true  -Failure $true)) { $errors++ }
if (-not (Set-AuditPolicy -Subcategory 'Other Logon/Logoff Events'    -Success $true  -Failure $true)) { $errors++ }
if (-not (Set-AuditPolicy -Subcategory 'Special Logon'                -Success $true  -Failure $false)) { $errors++ }

# Object Access
Write-Status "Configuring Object Access audit policies..."
if (-not (Set-AuditPolicy -Subcategory 'Detailed File Share'           -Success $false -Failure $true)) { $errors++ }
if (-not (Set-AuditPolicy -Subcategory 'File Share'                    -Success $true  -Failure $true)) { $errors++ }
if (-not (Set-AuditPolicy -Subcategory 'Other Object Access Events'    -Success $true  -Failure $true)) { $errors++ }
if (-not (Set-AuditPolicy -Subcategory 'Removable Storage'             -Success $true  -Failure $true)) { $errors++ }

# Policy Change
Write-Status "Configuring Policy Change audit policies..."
if (-not (Set-AuditPolicy -Subcategory 'Audit Policy Change'         -Success $true  -Failure $false)) { $errors++ }
if (-not (Set-AuditPolicy -Subcategory 'Authentication Policy Change' -Success $true  -Failure $false)) { $errors++ }
if (-not (Set-AuditPolicy -Subcategory 'Authorization Policy Change'  -Success $true  -Failure $false)) { $errors++ }
if (-not (Set-AuditPolicy -Subcategory 'MPSSVC Rule-Level Policy Change' -Success $true -Failure $true)) { $errors++ }
if (-not (Set-AuditPolicy -Subcategory 'Other Policy Change Events'   -Success $false -Failure $true)) { $errors++ }

# Privilege Use
Write-Status "Configuring Privilege Use audit policies..."
if (-not (Set-AuditPolicy -Subcategory 'Sensitive Privilege Use' -Success $true -Failure $true)) { $errors++ }

# System
Write-Status "Configuring System audit policies..."
if (-not (Set-AuditPolicy -Subcategory 'IPsec Driver'              -Success $true -Failure $true)) { $errors++ }
if (-not (Set-AuditPolicy -Subcategory 'Other System Events'       -Success $true -Failure $true)) { $errors++ }
if (-not (Set-AuditPolicy -Subcategory 'Security State Change'     -Success $true -Failure $false)) { $errors++ }
if (-not (Set-AuditPolicy -Subcategory 'Security System Extension' -Success $true -Failure $false)) { $errors++ }
if (-not (Set-AuditPolicy -Subcategory 'System Integrity'          -Success $true -Failure $true)) { $errors++ }

if ($errors -gt 0) {
    Write-Status "Audit policy remediation completed with $errors error(s). Review output above." 'WARN'
    exit 1
} else {
    Write-Status "Audit policy remediation complete. All policies configured successfully."
}
