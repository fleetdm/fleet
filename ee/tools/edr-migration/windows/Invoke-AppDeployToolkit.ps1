<#
.SYNOPSIS
    Grubhub EDR Migration — SentinelOne to CrowdStrike Falcon (Windows)

.DESCRIPTION
    PowerShell App Deployment Toolkit (PSADT) v4.1 deployment script that
    migrates Windows endpoints from SentinelOne to CrowdStrike Falcon.

    Designed for deployment via Fleet, SCCM, Intune, or GPO.

    Branded for Grubhub IT.

.NOTES
    PSADT Version: 4.1
    Author:        Grubhub IT
    Date:          2026-03-12
#>

# =============================================================================
# PSADT v4.1 Entry Point
# =============================================================================

[CmdletBinding()]
param (
    [Parameter(Mandatory = $false)]
    [ValidateSet('Install', 'Uninstall', 'Repair')]
    [String]$DeploymentType = 'Install',

    [Parameter(Mandatory = $false)]
    [ValidateSet('Interactive', 'Silent', 'NonInteractive')]
    [String]$DeployMode = 'Interactive',

    [Parameter(Mandatory = $false)]
    [Switch]$AllowRebootPassThru = $false,

    [Parameter(Mandatory = $false)]
    [Switch]$TerminalServerMode = $false
)

# =============================================================================
# CONFIGURATION — Set these variables before deployment
# =============================================================================

# Path to the CrowdStrike Falcon sensor installer (relative to .\Files\ or absolute)
$CrowdStrikeInstallerPath = ".\Files\CrowdStrike\WindowsSensor.exe"

# CrowdStrike Customer ID (CID) — obtain from your Falcon console
$CrowdStrikeCID = ""

# SentinelOne management passphrase/token — required for uninstall on managed endpoints
$SentinelOnePassphrase = ""

# Support contact info shown to end users on failure
$SupportEmail = "it-security@grubhub.com"
$SupportURL = "https://grubhub.service-now.com"

# SentinelOne paths (auto-detected but can be overridden)
$SentinelOneInstallDir = "C:\Program Files\SentinelOne\Sentinel Agent"
$SentinelOneUninstaller = "$SentinelOneInstallDir\uninstall.exe"

# Log directory
$LogDir = "C:\Logs\GrubhubIT"
$LogFile = "$LogDir\EDR_Migration.log"

# =============================================================================
# PSADT v4.1 INITIALIZATION
# =============================================================================

try {
    # Source the PSADT module — v4.1 uses the AppDeployToolkit subdirectory
    $adtPath = Join-Path -Path $PSScriptRoot -ChildPath 'AppDeployToolkit\AppDeployToolkitMain.ps1'
    if (Test-Path -Path $adtPath) {
        . $adtPath
    }
    else {
        # Fallback: try the v4.1 module import pattern
        $modulePath = Join-Path -Path $PSScriptRoot -ChildPath 'AppDeployToolkit\PSAppDeployToolkit'
        if (Test-Path -Path $modulePath) {
            Import-Module -Name $modulePath -Force
        }
        else {
            Write-Warning "PSADT framework not found. Running in standalone mode."
        }
    }
}
catch {
    Write-Warning "Failed to initialize PSADT framework: $_"
}

# =============================================================================
# PSADT v4.1 VARIABLES
# =============================================================================

$appVendor = 'Grubhub IT'
$appName = 'EDR Migration — SentinelOne to CrowdStrike Falcon'
$appVersion = '1.0.0'
$appArch = 'x64'
$appLang = 'EN'
$appRevision = '01'
$appScriptVersion = '1.0.0'
$appScriptDate = '2026-03-12'
$appScriptAuthor = 'Grubhub IT'

$installTitle = 'Grubhub IT — Endpoint Security Migration'

# =============================================================================
# HELPER FUNCTIONS
# =============================================================================

function Write-MigrationLog {
    <#
    .SYNOPSIS
        Write a timestamped log entry to the migration log file.
    #>
    [CmdletBinding()]
    param (
        [Parameter(Mandatory = $true)]
        [String]$Message,

        [Parameter(Mandatory = $false)]
        [ValidateSet('INFO', 'WARN', 'ERROR')]
        [String]$Level = 'INFO'
    )

    $timestamp = Get-Date -Format 'yyyy-MM-dd HH:mm:ss'
    $logEntry = "[$timestamp] [$Level] $Message"

    # Ensure log directory exists
    if (-not (Test-Path -Path $LogDir)) {
        New-Item -Path $LogDir -ItemType Directory -Force | Out-Null
    }

    Add-Content -Path $LogFile -Value $logEntry -Encoding UTF8

    # Also write to PSADT log if available
    try {
        if (Get-Command -Name 'Write-Log' -ErrorAction SilentlyContinue) {
            Write-Log -Message $Message -Severity ([int](@{ 'INFO' = 1; 'WARN' = 2; 'ERROR' = 3 }[$Level]))
        }
    }
    catch {
        # PSADT logging not available — already logged to file
    }
}

function Test-SentinelOneInstalled {
    <#
    .SYNOPSIS
        Check if SentinelOne is installed on the system.
    #>
    [CmdletBinding()]
    [OutputType([bool])]
    param()

    # Check 1: Service exists
    $service = Get-Service -Name 'SentinelAgent' -ErrorAction SilentlyContinue
    if ($service) {
        Write-MigrationLog "SentinelOne detected: service 'SentinelAgent' found (Status: $($service.Status))"
        return $true
    }

    # Check 2: Registry (64-bit)
    $regPath = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
    $s1Reg = Get-ItemProperty -Path $regPath -ErrorAction SilentlyContinue |
        Where-Object { $_.DisplayName -like '*SentinelOne*' -or $_.DisplayName -like '*Sentinel Agent*' }
    if ($s1Reg) {
        Write-MigrationLog "SentinelOne detected: registry entry found ($($s1Reg.DisplayName))"
        return $true
    }

    # Check 3: Registry (32-bit on 64-bit OS)
    $regPath32 = 'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
    $s1Reg32 = Get-ItemProperty -Path $regPath32 -ErrorAction SilentlyContinue |
        Where-Object { $_.DisplayName -like '*SentinelOne*' -or $_.DisplayName -like '*Sentinel Agent*' }
    if ($s1Reg32) {
        Write-MigrationLog "SentinelOne detected: 32-bit registry entry found ($($s1Reg32.DisplayName))"
        return $true
    }

    # Check 4: Executable exists
    if (Test-Path -Path "$SentinelOneInstallDir\SentinelAgent.exe") {
        Write-MigrationLog "SentinelOne detected: SentinelAgent.exe found"
        return $true
    }

    Write-MigrationLog "SentinelOne not detected on this system"
    return $false
}

function Test-CrowdStrikeInstalled {
    <#
    .SYNOPSIS
        Check if CrowdStrike Falcon is installed and running.
    #>
    [CmdletBinding()]
    [OutputType([bool])]
    param()

    $service = Get-Service -Name 'CSFalconService' -ErrorAction SilentlyContinue
    if ($service -and $service.Status -eq 'Running') {
        Write-MigrationLog "CrowdStrike Falcon detected: service 'CSFalconService' is running"
        return $true
    }

    return $false
}

function Uninstall-SentinelOneAgent {
    <#
    .SYNOPSIS
        Uninstall SentinelOne using available methods.
    .OUTPUTS
        [bool] True if uninstall succeeded, False otherwise.
    #>
    [CmdletBinding()]
    [OutputType([bool])]
    param()

    Write-MigrationLog "Starting SentinelOne uninstall process"

    # --- Method 1: Use the SentinelOne uninstaller executable ---
    if (Test-Path -Path $SentinelOneUninstaller) {
        Write-MigrationLog "Attempting uninstall via: $SentinelOneUninstaller"

        $uninstallArgs = '/q'
        if (-not [string]::IsNullOrWhiteSpace($SentinelOnePassphrase)) {
            $uninstallArgs = "/q /t $SentinelOnePassphrase"
        }

        try {
            if (Get-Command -Name 'Execute-Process' -ErrorAction SilentlyContinue) {
                $result = Execute-Process -Path $SentinelOneUninstaller `
                    -Parameters $uninstallArgs `
                    -WaitForMsiExec:$true `
                    -PassThru `
                    -IgnoreExitCodes '3010'
                $exitCode = $result.ExitCode
            }
            else {
                $proc = Start-Process -FilePath $SentinelOneUninstaller `
                    -ArgumentList $uninstallArgs `
                    -Wait -PassThru -NoNewWindow
                $exitCode = $proc.ExitCode
            }

            if ($exitCode -eq 0 -or $exitCode -eq 3010) {
                Write-MigrationLog "SentinelOne uninstaller completed (exit code: $exitCode)"
                if ($exitCode -eq 3010) {
                    Write-MigrationLog -Level 'WARN' "Reboot may be required to complete SentinelOne removal"
                }
                return $true
            }
            else {
                Write-MigrationLog -Level 'ERROR' "SentinelOne uninstaller failed (exit code: $exitCode)"
            }
        }
        catch {
            Write-MigrationLog -Level 'ERROR' "SentinelOne uninstaller threw an exception: $_"
        }
    }

    # --- Method 2: MSI uninstall via registry ---
    Write-MigrationLog "Attempting MSI-based uninstall"

    $regPaths = @(
        'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
        'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
    )

    foreach ($regPath in $regPaths) {
        $s1Entry = Get-ItemProperty -Path $regPath -ErrorAction SilentlyContinue |
            Where-Object { $_.DisplayName -like '*SentinelOne*' -or $_.DisplayName -like '*Sentinel Agent*' } |
            Select-Object -First 1

        if ($s1Entry -and $s1Entry.UninstallString) {
            $uninstallCmd = $s1Entry.UninstallString
            Write-MigrationLog "Found uninstall string: $uninstallCmd"

            # Extract MSI product code if present
            if ($uninstallCmd -match '\{[0-9A-Fa-f-]+\}') {
                $productCode = $Matches[0]
                Write-MigrationLog "MSI Product Code: $productCode"

                $msiArgs = "/x $productCode /qn /norestart"
                if (-not [string]::IsNullOrWhiteSpace($SentinelOnePassphrase)) {
                    $msiArgs += " PASSPHRASE=$SentinelOnePassphrase"
                }

                try {
                    if (Get-Command -Name 'Execute-MSI' -ErrorAction SilentlyContinue) {
                        Execute-MSI -Action 'Uninstall' -Path $productCode -Parameters "PASSPHRASE=$SentinelOnePassphrase"
                    }
                    else {
                        $proc = Start-Process -FilePath 'msiexec.exe' `
                            -ArgumentList $msiArgs `
                            -Wait -PassThru -NoNewWindow
                        if ($proc.ExitCode -eq 0 -or $proc.ExitCode -eq 3010) {
                            Write-MigrationLog "MSI uninstall completed (exit code: $($proc.ExitCode))"
                            return $true
                        }
                        else {
                            Write-MigrationLog -Level 'ERROR' "MSI uninstall failed (exit code: $($proc.ExitCode))"
                        }
                    }
                }
                catch {
                    Write-MigrationLog -Level 'ERROR' "MSI uninstall exception: $_"
                }
            }
        }
    }

    # --- Method 3: SentinelCtl ---
    $sentinelCtl = "$SentinelOneInstallDir\sentinelctl.exe"
    if (Test-Path -Path $sentinelCtl) {
        Write-MigrationLog "Attempting uninstall via sentinelctl"
        try {
            $ctlArgs = 'uninstall'
            if (-not [string]::IsNullOrWhiteSpace($SentinelOnePassphrase)) {
                $ctlArgs = "uninstall --passphrase $SentinelOnePassphrase"
            }

            $proc = Start-Process -FilePath $sentinelCtl `
                -ArgumentList $ctlArgs `
                -Wait -PassThru -NoNewWindow

            if ($proc.ExitCode -eq 0) {
                Write-MigrationLog "sentinelctl uninstall completed successfully"
                return $true
            }
            else {
                Write-MigrationLog -Level 'ERROR' "sentinelctl uninstall failed (exit code: $($proc.ExitCode))"
            }
        }
        catch {
            Write-MigrationLog -Level 'ERROR' "sentinelctl exception: $_"
        }
    }

    Write-MigrationLog -Level 'ERROR' "All SentinelOne uninstall methods failed"
    return $false
}

function Install-CrowdStrikeFalcon {
    <#
    .SYNOPSIS
        Install CrowdStrike Falcon sensor and configure with CID.
    .OUTPUTS
        [bool] True if install succeeded, False otherwise.
    #>
    [CmdletBinding()]
    [OutputType([bool])]
    param()

    Write-MigrationLog "Starting CrowdStrike Falcon installation"

    # Resolve the installer path
    $installerPath = $CrowdStrikeInstallerPath
    if (-not [System.IO.Path]::IsPathRooted($installerPath)) {
        $installerPath = Join-Path -Path $PSScriptRoot -ChildPath $installerPath
    }

    if (-not (Test-Path -Path $installerPath)) {
        Write-MigrationLog -Level 'ERROR' "CrowdStrike installer not found at: $installerPath"
        return $false
    }

    Write-MigrationLog "CrowdStrike installer path: $installerPath"

    # Build installer arguments
    $installArgs = "/install /quiet /norestart CID=$CrowdStrikeCID"

    # Determine installer type and execute
    $extension = [System.IO.Path]::GetExtension($installerPath).ToLower()

    try {
        switch ($extension) {
            '.exe' {
                Write-MigrationLog "Running EXE installer with args: $installArgs"

                if (Get-Command -Name 'Execute-Process' -ErrorAction SilentlyContinue) {
                    $result = Execute-Process -Path $installerPath `
                        -Parameters $installArgs `
                        -WaitForMsiExec:$true `
                        -PassThru `
                        -IgnoreExitCodes '3010'
                    $exitCode = $result.ExitCode
                }
                else {
                    $proc = Start-Process -FilePath $installerPath `
                        -ArgumentList $installArgs `
                        -Wait -PassThru -NoNewWindow
                    $exitCode = $proc.ExitCode
                }
            }
            '.msi' {
                Write-MigrationLog "Running MSI installer"
                $msiArgs = "/i `"$installerPath`" /qn /norestart CID=$CrowdStrikeCID"

                if (Get-Command -Name 'Execute-MSI' -ErrorAction SilentlyContinue) {
                    Execute-MSI -Action 'Install' -Path $installerPath -Parameters "CID=$CrowdStrikeCID"
                    $exitCode = 0
                }
                else {
                    $proc = Start-Process -FilePath 'msiexec.exe' `
                        -ArgumentList $msiArgs `
                        -Wait -PassThru -NoNewWindow
                    $exitCode = $proc.ExitCode
                }
            }
            default {
                Write-MigrationLog -Level 'ERROR' "Unsupported installer type: $extension"
                return $false
            }
        }

        if ($exitCode -eq 0 -or $exitCode -eq 3010) {
            Write-MigrationLog "CrowdStrike installer completed (exit code: $exitCode)"
            if ($exitCode -eq 3010) {
                Write-MigrationLog -Level 'WARN' "Reboot may be required to complete CrowdStrike installation"
            }
        }
        else {
            Write-MigrationLog -Level 'ERROR' "CrowdStrike installer failed (exit code: $exitCode)"
            return $false
        }
    }
    catch {
        Write-MigrationLog -Level 'ERROR' "CrowdStrike installer exception: $_"
        return $false
    }

    # Wait for the service to start (up to 60 seconds)
    Write-MigrationLog "Waiting for CSFalconService to start..."
    $maxWait = 60
    $waited = 0
    $serviceRunning = $false

    while ($waited -lt $maxWait) {
        $service = Get-Service -Name 'CSFalconService' -ErrorAction SilentlyContinue
        if ($service -and $service.Status -eq 'Running') {
            $serviceRunning = $true
            break
        }
        Start-Sleep -Seconds 5
        $waited += 5
        Write-MigrationLog "Waiting for CSFalconService... ($waited/$maxWait seconds)"
    }

    if ($serviceRunning) {
        Write-MigrationLog "CSFalconService is running"
    }
    else {
        Write-MigrationLog -Level 'WARN' "CSFalconService did not start within $maxWait seconds — it may still be initializing"
    }

    return $true
}

function Test-SentinelOneRemoved {
    <#
    .SYNOPSIS
        Verify that SentinelOne has been fully removed.
    #>
    [CmdletBinding()]
    [OutputType([bool])]
    param()

    $service = Get-Service -Name 'SentinelAgent' -ErrorAction SilentlyContinue
    if ($service) {
        Write-MigrationLog -Level 'WARN' "SentinelAgent service still exists (Status: $($service.Status))"
        return $false
    }

    if (Test-Path -Path "$SentinelOneInstallDir\SentinelAgent.exe") {
        Write-MigrationLog -Level 'WARN' "SentinelAgent.exe still exists on disk"
        return $false
    }

    Write-MigrationLog "SentinelOne removal verified — no traces found"
    return $true
}

# =============================================================================
# PSADT v4.1 DEPLOYMENT PHASES
# =============================================================================

# ─────────────────────────────────────────────────────────────────────────────
# PRE-INSTALLATION
# ─────────────────────────────────────────────────────────────────────────────

Write-MigrationLog "========================================="
Write-MigrationLog "Grubhub EDR Migration — Starting"
Write-MigrationLog "========================================="
Write-MigrationLog "Hostname: $env:COMPUTERNAME"
Write-MigrationLog "OS: $([System.Environment]::OSVersion.VersionString)"
Write-MigrationLog "User: $env:USERNAME"
Write-MigrationLog "Deployment Type: $DeploymentType"
Write-MigrationLog "Deployment Mode: $DeployMode"

if ($DeploymentType -eq 'Install') {

    # --- Show welcome dialog ---
    try {
        if (Get-Command -Name 'Show-InstallationWelcome' -ErrorAction SilentlyContinue) {
            Show-InstallationWelcome `
                -CloseApps 'SentinelAgent,SentinelUI' `
                -CloseAppsCountdown 300 `
                -CheckDiskSpace `
                -PersistPrompt `
                -BlockExecution
        }
    }
    catch {
        Write-MigrationLog -Level 'WARN' "Show-InstallationWelcome not available: $_"
    }

    # --- Pre-flight: Check if CrowdStrike is already installed ---
    if (Test-CrowdStrikeInstalled) {
        Write-MigrationLog "CrowdStrike Falcon is already installed and running. No migration needed."
        try {
            if (Get-Command -Name 'Show-InstallationPrompt' -ErrorAction SilentlyContinue) {
                Show-InstallationPrompt `
                    -Message "CrowdStrike Falcon is already installed and running on this PC.`n`nNo migration is needed.`n`n— Grubhub IT" `
                    -ButtonRightText 'OK' `
                    -Icon 'Information'
            }
        }
        catch {
            Write-MigrationLog -Level 'WARN' "Could not show already-installed dialog: $_"
        }
        Write-MigrationLog "Exiting — no action required"
        exit 0
    }

    # --- Pre-flight: Check SentinelOne status ---
    $sentinelOnePresent = Test-SentinelOneInstalled

    # --- Pre-flight: Validate configuration ---
    if ([string]::IsNullOrWhiteSpace($CrowdStrikeCID)) {
        Write-MigrationLog -Level 'ERROR' "CrowdStrikeCID is not configured. Aborting."
        try {
            if (Get-Command -Name 'Show-InstallationPrompt' -ErrorAction SilentlyContinue) {
                Show-InstallationPrompt `
                    -Message "Migration cannot proceed: CrowdStrike CID is not configured.`n`nPlease contact Grubhub IT.`n`nEmail: $SupportEmail`nPortal: $SupportURL" `
                    -ButtonRightText 'OK' `
                    -Icon 'Error'
            }
        }
        catch { }
        exit 1
    }

    # --- Balloon notification ---
    try {
        if (Get-Command -Name 'Show-BalloonTip' -ErrorAction SilentlyContinue) {
            Show-BalloonTip `
                -BalloonTipIcon 'Info' `
                -BalloonTipTitle 'Grubhub IT' `
                -BalloonTipText 'Endpoint security migration starting. This will take approximately 5 minutes.'
        }
    }
    catch {
        Write-MigrationLog -Level 'WARN' "Could not show balloon notification: $_"
    }

    # ─────────────────────────────────────────────────────────────────────────
    # INSTALLATION
    # ─────────────────────────────────────────────────────────────────────────

    # --- Show progress dialog ---
    try {
        if (Get-Command -Name 'Show-InstallationProgress' -ErrorAction SilentlyContinue) {
            Show-InstallationProgress `
                -StatusMessage "Grubhub IT is upgrading your endpoint protection.`n`nThis may take a few minutes. Please do not restart your computer." `
                -TopMost $true
        }
    }
    catch {
        Write-MigrationLog -Level 'WARN' "Show-InstallationProgress not available: $_"
    }

    $migrationFailed = $false
    $errorDetails = ""

    # --- Step 1: Uninstall SentinelOne ---
    if ($sentinelOnePresent) {
        Write-MigrationLog "Phase: Uninstalling SentinelOne"

        try {
            if (Get-Command -Name 'Show-InstallationProgress' -ErrorAction SilentlyContinue) {
                Show-InstallationProgress `
                    -StatusMessage "Grubhub IT: Removing SentinelOne...`n`nPlease wait while the previous security software is removed."
            }
        }
        catch { }

        $uninstallResult = Uninstall-SentinelOneAgent

        if (-not $uninstallResult) {
            Write-MigrationLog -Level 'ERROR' "SentinelOne uninstall failed"
            $migrationFailed = $true
            $errorDetails = "Failed to uninstall SentinelOne. The previous security software could not be removed."
        }
        else {
            # Allow time for cleanup
            Write-MigrationLog "Waiting for SentinelOne cleanup..."
            Start-Sleep -Seconds 10

            # Verify removal
            if (-not (Test-SentinelOneRemoved)) {
                Write-MigrationLog -Level 'WARN' "SentinelOne may not be fully removed — proceeding with CrowdStrike install"
            }
        }
    }
    else {
        Write-MigrationLog "SentinelOne not present — skipping uninstall"
    }

    # --- Step 2: Install CrowdStrike Falcon ---
    if (-not $migrationFailed) {
        Write-MigrationLog "Phase: Installing CrowdStrike Falcon"

        try {
            if (Get-Command -Name 'Show-InstallationProgress' -ErrorAction SilentlyContinue) {
                Show-InstallationProgress `
                    -StatusMessage "Grubhub IT: Installing CrowdStrike Falcon...`n`nPlease wait while the new security software is installed and configured."
            }
        }
        catch { }

        $installResult = Install-CrowdStrikeFalcon

        if (-not $installResult) {
            Write-MigrationLog -Level 'ERROR' "CrowdStrike Falcon installation failed"
            $migrationFailed = $true
            $errorDetails = "Failed to install CrowdStrike Falcon. The new security software could not be installed."
        }
    }

    # ─────────────────────────────────────────────────────────────────────────
    # POST-INSTALLATION
    # ─────────────────────────────────────────────────────────────────────────

    # Close the progress dialog
    try {
        if (Get-Command -Name 'Close-InstallationProgress' -ErrorAction SilentlyContinue) {
            Close-InstallationProgress
        }
    }
    catch { }

    if ($migrationFailed) {
        Write-MigrationLog -Level 'ERROR' "Migration FAILED: $errorDetails"

        try {
            if (Get-Command -Name 'Show-InstallationPrompt' -ErrorAction SilentlyContinue) {
                Show-InstallationPrompt `
                    -Message "Endpoint Security Migration Failed`n`n$errorDetails`n`nPlease contact Grubhub IT for assistance:`n`nEmail: $SupportEmail`nPortal: $SupportURL`n`nPlease include your computer name ($env:COMPUTERNAME) when contacting support." `
                    -ButtonRightText 'OK' `
                    -Icon 'Error'
            }
        }
        catch {
            Write-MigrationLog -Level 'WARN' "Could not show failure dialog: $_"
        }

        Write-MigrationLog "========================================="
        Write-MigrationLog "Grubhub EDR Migration — FAILED"
        Write-MigrationLog "========================================="
        exit 1
    }

    # --- Verify CrowdStrike Falcon is running ---
    Write-MigrationLog "Phase: Post-installation verification"

    $csRunning = $false
    $service = Get-Service -Name 'CSFalconService' -ErrorAction SilentlyContinue
    if ($service -and $service.Status -eq 'Running') {
        $csRunning = $true
        Write-MigrationLog "Verification: CSFalconService is running"
    }
    else {
        Write-MigrationLog -Level 'WARN' "Verification: CSFalconService is not running — may still be initializing"
    }

    # --- Show completion dialog ---
    $completionStatus = if ($csRunning) { "running and active" } else { "installed (service is still initializing)" }

    try {
        if (Get-Command -Name 'Show-InstallationPrompt' -ErrorAction SilentlyContinue) {
            Show-InstallationPrompt `
                -Message "Migration Complete!`n`nYour PC is now protected by CrowdStrike Falcon ($completionStatus).`n`nNo further action is needed on your part.`n`nThank you for your patience!`n`n— Grubhub IT" `
                -ButtonRightText 'OK' `
                -Icon 'Information'
        }
    }
    catch {
        Write-MigrationLog -Level 'WARN' "Could not show completion dialog: $_"
    }

    # --- Balloon notification ---
    try {
        if (Get-Command -Name 'Show-BalloonTip' -ErrorAction SilentlyContinue) {
            Show-BalloonTip `
                -BalloonTipIcon 'Info' `
                -BalloonTipTitle 'Grubhub IT' `
                -BalloonTipText 'Endpoint security migration complete. Your PC is now protected by CrowdStrike Falcon.'
        }
    }
    catch { }

    Write-MigrationLog "========================================="
    Write-MigrationLog "Grubhub EDR Migration — Completed Successfully"
    Write-MigrationLog "========================================="
}
elseif ($DeploymentType -eq 'Uninstall') {
    # ─────────────────────────────────────────────────────────────────────────
    # UNINSTALL MODE — Roll back: remove CrowdStrike, reinstall SentinelOne
    # ─────────────────────────────────────────────────────────────────────────

    Write-MigrationLog "Uninstall mode: This would roll back the migration (not implemented)"
    Write-MigrationLog "To roll back, manually uninstall CrowdStrike Falcon and reinstall SentinelOne"

    try {
        if (Get-Command -Name 'Show-InstallationPrompt' -ErrorAction SilentlyContinue) {
            Show-InstallationPrompt `
                -Message "Rollback is not automated.`n`nTo roll back the EDR migration, please contact Grubhub IT.`n`nEmail: $SupportEmail`nPortal: $SupportURL" `
                -ButtonRightText 'OK' `
                -Icon 'Information'
        }
    }
    catch { }
}
elseif ($DeploymentType -eq 'Repair') {
    # ─────────────────────────────────────────────────────────────────────────
    # REPAIR MODE — Verify CrowdStrike is healthy
    # ─────────────────────────────────────────────────────────────────────────

    Write-MigrationLog "Repair mode: Verifying CrowdStrike Falcon installation"

    if (Test-CrowdStrikeInstalled) {
        Write-MigrationLog "CrowdStrike Falcon is installed and running — no repair needed"
    }
    else {
        Write-MigrationLog -Level 'WARN' "CrowdStrike Falcon is not running — attempting reinstall"
        $installResult = Install-CrowdStrikeFalcon
        if ($installResult) {
            Write-MigrationLog "CrowdStrike Falcon repair completed"
        }
        else {
            Write-MigrationLog -Level 'ERROR' "CrowdStrike Falcon repair failed"
            exit 1
        }
    }
}

exit 0
