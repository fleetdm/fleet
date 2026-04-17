#Requires -RunAsAdministrator
<#
.SYNOPSIS
    CIS Windows 11 Enterprise Benchmark v4.0.0 - User Rights Assignment Remediation

.DESCRIPTION
    Configures user rights assignments (privileges) to meet CIS Windows 11 Enterprise Benchmark v4.0.0
    requirements using secedit. This script is idempotent - safe to run multiple times.

.NOTES
    CIS Benchmark Controls Covered (Section 2.2):
    - 2.2.1  Access Credential Manager as a trusted caller    -> (empty)
    - 2.2.2  Access this computer from the network           -> Administrators, Remote Desktop Users
    - 2.2.3  Act as part of the operating system             -> (empty)
    - 2.2.5  Allow log on locally                            -> Administrators, Users
    - 2.2.7  Back up files and directories                   -> Administrators
    - 2.2.8  Change the system time                          -> Administrators, LOCAL SERVICE
    - 2.2.11 Create a token object                           -> (empty)
    - 2.2.12 Create global objects                           -> Administrators, LOCAL SERVICE, NETWORK SERVICE, SERVICE
    - 2.2.13 Create permanent shared objects                 -> (empty)
    - 2.2.14 Create symbolic links                           -> Administrators
    - 2.2.15 Debug programs                                  -> Administrators
    - 2.2.16 Deny access this computer from the network      -> Guests, Local account
    - 2.2.19 Deny log on locally                             -> Guests
    - 2.2.20 Deny log on through Remote Desktop Services     -> Guests, Local account
    - 2.2.21 Enable computer and user accounts for delegation -> (empty)
    - 2.2.22 Force shutdown from a remote system             -> Administrators
    - 2.2.23 Generate security audits                        -> LOCAL SERVICE, NETWORK SERVICE
    - 2.2.24 Impersonate a client after authentication       -> Administrators, LOCAL SERVICE, NETWORK SERVICE, SERVICE
    - 2.2.25 Increase scheduling priority                    -> Administrators
    - 2.2.26 Load and unload device drivers                  -> Administrators
    - 2.2.27 Lock pages in memory                            -> (empty)
    - 2.2.30 Manage auditing and security log                -> Administrators
    - 2.2.31 Modify an object label                          -> (empty)
    - 2.2.32 Modify firmware environment values              -> Administrators
    - 2.2.34 Profile single process                          -> Administrators
    - 2.2.37 Restore files and directories                   -> Administrators
    - 2.2.39 Take ownership of files or other objects        -> Administrators

    SID Reference used in secedit [Privilege Rights]:
    - *S-1-5-6    = SERVICE
    - *S-1-5-19   = LOCAL SERVICE
    - *S-1-5-20   = NETWORK SERVICE
    - *S-1-5-32-544 = Administrators
    - *S-1-5-32-545 = Users
    - *S-1-5-32-546 = Guests
    - *S-1-5-32-555 = Remote Desktop Users

    Requires: Administrator privileges
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

Write-Status "Starting CIS user rights assignment remediation"

$cfgFile = Join-Path $env:TEMP "secpol_ura.cfg"
$sdbFile = Join-Path $env:TEMP "secedit_ura.sdb"
$logFile = Join-Path $env:TEMP "secedit_ura.log"

# Define the desired user rights (secedit format: privilege = SID list)
# Empty value means "No One" - the right is granted to no accounts
$userRights = [ordered]@{
    # CIS 2.2.1: No One
    'SeTrustedCredManAccessPrivilege'   = ''
    # CIS 2.2.2: Administrators, Remote Desktop Users
    'SeNetworkLogonRight'               = '*S-1-5-32-544,*S-1-5-32-555'
    # CIS 2.2.3: No One
    'SeTcbPrivilege'                    = ''
    # CIS 2.2.5: Administrators, Users
    'SeInteractiveLogonRight'           = '*S-1-5-32-544,*S-1-5-32-545'
    # CIS 2.2.7: Administrators
    'SeBackupPrivilege'                 = '*S-1-5-32-544'
    # CIS 2.2.8: Administrators, LOCAL SERVICE
    'SeSystemtimePrivilege'             = '*S-1-5-32-544,*S-1-5-19'
    # CIS 2.2.11: No One
    'SeCreateTokenPrivilege'            = ''
    # CIS 2.2.12: Administrators, LOCAL SERVICE, NETWORK SERVICE, SERVICE
    'SeCreateGlobalPrivilege'           = '*S-1-5-32-544,*S-1-5-19,*S-1-5-20,*S-1-5-6'
    # CIS 2.2.13: No One
    'SeCreatePermanentPrivilege'        = ''
    # CIS 2.2.14: Administrators
    'SeCreateSymbolicLinkPrivilege'     = '*S-1-5-32-544'
    # CIS 2.2.15: Administrators
    'SeDebugPrivilege'                  = '*S-1-5-32-544'
    # CIS 2.2.16: Guests, Local account
    'SeDenyNetworkLogonRight'           = '*S-1-5-32-546,*S-1-5-113'
    # CIS 2.2.19: Guests
    'SeDenyInteractiveLogonRight'       = '*S-1-5-32-546'
    # CIS 2.2.20: Guests, Local account
    'SeDenyRemoteInteractiveLogonRight' = '*S-1-5-32-546,*S-1-5-113'
    # CIS 2.2.21: No One
    'SeEnableDelegationPrivilege'       = ''
    # CIS 2.2.22: Administrators
    'SeRemoteShutdownPrivilege'         = '*S-1-5-32-544'
    # CIS 2.2.23: LOCAL SERVICE, NETWORK SERVICE
    'SeAuditPrivilege'                  = '*S-1-5-19,*S-1-5-20'
    # CIS 2.2.24: Administrators, LOCAL SERVICE, NETWORK SERVICE, SERVICE
    'SeImpersonatePrivilege'            = '*S-1-5-32-544,*S-1-5-19,*S-1-5-20,*S-1-5-6'
    # CIS 2.2.25: Administrators
    'SeIncreaseBasePriorityPrivilege'   = '*S-1-5-32-544'
    # CIS 2.2.26: Administrators
    'SeLoadDriverPrivilege'             = '*S-1-5-32-544'
    # CIS 2.2.27: No One
    'SeLockMemoryPrivilege'             = ''
    # CIS 2.2.30: Administrators
    'SeSecurityPrivilege'               = '*S-1-5-32-544'
    # CIS 2.2.31: No One
    'SeRelabelPrivilege'                = ''
    # CIS 2.2.32: Administrators
    'SeSystemEnvironmentPrivilege'      = '*S-1-5-32-544'
    # CIS 2.2.34: Administrators
    'SeProfileSingleProcessPrivilege'   = '*S-1-5-32-544'
    # CIS 2.2.37: Administrators
    'SeRestorePrivilege'                = '*S-1-5-32-544'
    # CIS 2.2.39: Administrators
    'SeTakeOwnershipPrivilege'          = '*S-1-5-32-544'
}

try {
    Write-Status "Exporting current security policy..."
    $null = secedit /export /cfg $cfgFile /quiet
    if ($LASTEXITCODE -ne 0) {
        throw "secedit /export failed with exit code $LASTEXITCODE"
    }

    $content = Get-Content $cfgFile -Raw

    # Ensure [Privilege Rights] section exists
    if ($content -notmatch '\[Privilege Rights\]') {
        $content += "`r`n[Privilege Rights]`r`n"
    }

    # Update each user right
    foreach ($right in $userRights.GetEnumerator()) {
        $key   = $right.Key
        $value = $right.Value

        if ($content -match "(?m)^$key\s*=") {
            # Replace existing entry
            $content = $content -replace "(?m)^$key\s*=.*", "$key = $value"
        } else {
            # Add new entry in [Privilege Rights] section
            $content = $content -replace '(\[Privilege Rights\])', "`$1`r`n$key = $value"
        }
        Write-Status "  Set $key = $value"
    }

    Set-Content -Path $cfgFile -Value $content -Encoding Unicode

    Write-Status "Applying user rights configuration..."
    $null = secedit /configure /db $sdbFile /cfg $cfgFile /areas USER_RIGHTS /log $logFile /quiet
    if ($LASTEXITCODE -ne 0) {
        Write-Status "secedit /configure exit code: $LASTEXITCODE. Check $logFile for details." 'WARN'
    } else {
        Write-Status "User rights applied successfully."
    }

} catch {
    Write-Status "ERROR: $_" 'ERROR'
    exit 1
} finally {
    @($cfgFile, $sdbFile, $logFile) | ForEach-Object {
        if (Test-Path $_) { Remove-Item $_ -Force -ErrorAction SilentlyContinue }
    }
}

Write-Status "User rights assignment remediation complete."
