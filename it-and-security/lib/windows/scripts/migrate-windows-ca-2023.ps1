<#
.SYNOPSIS
    Migrates a Windows host to Secure Boot CA 2023 trust chain (KB5025885).

.DESCRIPTION
    Closes CVE-2023-24932 / BlackLotus boot manager swap vulnerability and
    ensures continued Secure Boot servicing past PCA 2011 expiry (June 2026).

    Verifies completion by checking BOTH:
      - File signatures on disk (bootmgfw.efi, winload.efi, winresume.efi)
      - Registry servicing state machine (UEFICA2023Status, Error, Capable)

    Idempotent and conservative:
      - Skips if all 3 boot binaries are already signed by CA 2023
      - Respects in-progress workflows (does not retrigger)
      - Bails if errored state detected (does not retry blindly)
      - Bails if prerequisites missing (Secure Boot off, CU too old)

.OUTPUTS
    Structured key:value output to stdout for log capture / parsing.

.NOTES
    Exit codes:
      0 = Migration complete, in progress, or just triggered (no action needed)
      2 = Secure Boot not enabled in firmware (firmware change required)
      3 = Cumulative update too old (run Windows Update first)
      4 = Errored state (UEFICA2023Error != 0); manual investigation required
      5 = Reboot pending; reboot to advance migration
      6 = Boot file missing or signature unreadable; cannot proceed with remediation

    References:
      KB5025885: https://support.microsoft.com/en-us/topic/41a975df-beb2-40c1-99a3-b3ff139f832d
      MS Secure Boot Playbook (Feb 2026):
      https://techcommunity.microsoft.com/blog/windows-itpro-blog/secure-boot-playbook-for-certificates-expiring-in-2026/4469235
#>

$ErrorActionPreference = 'Stop'

function Write-State {
    param([string]$Label, [string]$Value)
    Write-Output ("{0,-30} : {1}" -f $Label, $Value)
}

Write-Output "=== Windows Secure Boot CA 2023 migration ==="
Write-Output ""

# --- Preflight: Secure Boot enabled ---
$secureBootEnabled = $false
try { $secureBootEnabled = Confirm-SecureBootUEFI } catch {}
if (-not $secureBootEnabled) {
    Write-Output "FAIL: Secure Boot not enabled in firmware. Enable before running."
    exit 2
}
Write-State "Secure Boot" "enabled"

# --- Preflight: cumulative update current enough ---
$secBootTask = Get-ScheduledTask -TaskPath "\Microsoft\Windows\PI\" -TaskName "Secure-Boot-Update" -ErrorAction SilentlyContinue
if (-not $secBootTask) {
    Write-Output "FAIL: \Microsoft\Windows\PI\Secure-Boot-Update task missing."
    Write-Output "      Cumulative update too old. Install latest CU and retry."
    exit 3
}
Write-State "Secure-Boot-Update task" "present"

# --- Inspect file signatures ---
Write-Output ""
Write-Output "--- File signatures ---"

$bootFiles = @(
    "$env:SystemRoot\Boot\EFI\bootmgfw.efi",
    "$env:SystemRoot\System32\winload.efi",
    "$env:SystemRoot\System32\winresume.efi"
)

$migratedCount = 0
$missingCount  = 0
foreach ($f in $bootFiles) {
    $name = Split-Path $f -Leaf
    if (-not (Test-Path $f)) {
        Write-State $name "MISSING"
        $missingCount++
        continue
    }
    $issuer = $null
    try { $issuer = (Get-AuthenticodeSignature $f).SignerCertificate.Issuer } catch {}
    if (-not $issuer) {
        Write-State $name "signature unreadable"
    } elseif ($issuer -match 'Windows UEFI CA 2023') {
        Write-State $name "CA 2023 (migrated)"
        $migratedCount++
    } elseif ($issuer -match 'Production PCA 2011') {
        Write-State $name "PCA 2011 (not migrated)"
    } else {
        Write-State $name "unknown issuer: $issuer"
    }
}

# --- Early exit if inspection incomplete ---
if ($missingCount -gt 0) {
    Write-Output ""
    Write-Output "FAIL: $missingCount boot file(s) missing. Cannot proceed with remediation."
    Write-State "State" "inspection_incomplete_missing_files"
    exit 6
}

# Check for any unreadable signatures
$unreadableCount = 0
foreach ($f in $bootFiles) {
    if (Test-Path $f) {
        $issuer = $null
        try { $issuer = (Get-AuthenticodeSignature $f).SignerCertificate.Issuer } catch {}
        if (-not $issuer) {
            $unreadableCount++
        }
    }
}
if ($unreadableCount -gt 0) {
    Write-Output ""
    Write-Output "FAIL: $unreadableCount boot file(s) have unreadable signatures. Cannot proceed with remediation."
    Write-State "State" "inspection_incomplete_unreadable_signatures"
    exit 6
}

# --- Inspect registry servicing state machine ---
Write-Output ""
Write-Output "--- Registry servicing state ---"

$servicing  = "HKLM:\SYSTEM\CurrentControlSet\Control\SecureBoot\Servicing"
$secureboot = "HKLM:\SYSTEM\CurrentControlSet\Control\Secureboot"

$regStatus    = (Get-ItemProperty -Path $servicing  -Name "UEFICA2023Status"         -ErrorAction SilentlyContinue).UEFICA2023Status
$regCapable   = (Get-ItemProperty -Path $servicing  -Name "WindowsUEFICA2023Capable" -ErrorAction SilentlyContinue).WindowsUEFICA2023Capable
$regError     = (Get-ItemProperty -Path $servicing  -Name "UEFICA2023Error"          -ErrorAction SilentlyContinue).UEFICA2023Error
$regAvailable = (Get-ItemProperty -Path $secureboot -Name "AvailableUpdates"         -ErrorAction SilentlyContinue).AvailableUpdates

$availableStr = if ($null -ne $regAvailable) { "0x{0:X4} ({1})" -f $regAvailable, $regAvailable } else { "(not set)" }

$statusStr  = if ($regStatus)              { $regStatus }              else { "(not set)" }
$capableStr = if ($null -ne $regCapable)   { $regCapable.ToString() }  else { "(not set)" }
$errorStr   = if ($null -ne $regError)     { $regError.ToString() }    else { "(not set)" }

Write-State "UEFICA2023Status"         $statusStr
Write-State "WindowsUEFICA2023Capable" $capableStr
Write-State "UEFICA2023Error"          $errorStr
Write-State "AvailableUpdates"         $availableStr

# --- Decision: idempotency, conservative paths ---
Write-Output ""
Write-Output "--- Decision ---"

# Fully migrated: all 3 files on CA 2023, no error
if ($migratedCount -eq 3 -and (-not $regError -or $regError -eq 0)) {
    Write-Output "OK: Fully migrated. All boot binaries signed by CA 2023."
    exit 0
}

# Errored state: do not retry
if ($regError -and $regError -ne 0) {
    Write-Output "FAIL: UEFICA2023Error = $regError. Manual investigation required."
    Write-Output "      Check Event Viewer > System for Event ID 1801 (DB update blocked)"
    Write-Output "      or 1803 (no PK-signed KEK)."
    exit 4
}

# Firmware servicing reports Updated — done at firmware level even if OS-side files lag
if ($regStatus -eq 'Updated') {
    Write-Output "OK: Servicing reports Updated (capable = $regCapable)."
    Write-Output "    Firmware-level migration complete. OS-side files may show PCA 2011"
    Write-Output "    until the next Windows Update cycle refreshes the staging copy."
    Write-Output "    Run verify-windows-ca-2023.ps1 to confirm firmware DB / ESP state."
    exit 0
}

# In-flight: leave alone
if ($regStatus -eq 'InProgress') {
    Write-Output "OK: Migration in progress (UEFICA2023Status = InProgress)."
    Write-Output "    Files migrated: $migratedCount / 3. Reboot may be required to advance."
    exit 0
}

# Reboot pending
if ($regAvailable -eq 0x4100) {
    Write-Output "WAIT: Reboot pending (AvailableUpdates = 0x4100)."
    Write-Output "      Reboot to advance migration. Re-run script after reboot."
    exit 5
}

# Trigger full deployment
if ($migratedCount -gt 0) {
    Write-Output "Partial migration detected ($migratedCount / 3 files on CA 2023). Retriggering."
} else {
    Write-Output "Triggering CA 2023 migration (AvailableUpdates = 0x5944) ..."
}
Set-ItemProperty -Path $secureboot -Name "AvailableUpdates" -Value 0x5944 -Type DWord -Force
$taskStarted = $null
try {
    Start-ScheduledTask -TaskPath "\Microsoft\Windows\PI\" -TaskName "Secure-Boot-Update"
    $taskStarted = Get-ScheduledTask -TaskPath "\Microsoft\Windows\PI\" -TaskName "Secure-Boot-Update" -ErrorAction SilentlyContinue
} catch {}
if (-not $taskStarted) {
    Write-Output "FAIL: Could not start Secure-Boot-Update task."
    exit 3
}

Write-Output "OK: Migration triggered. Reboot required (sometimes two)."
Write-Output "    Re-run this script after each reboot to confirm progress."
exit 0
