<#
.SYNOPSIS
    Verifies Windows Secure Boot CA 2023 migration at firmware level.

.DESCRIPTION
    Reads firmware-level state that osquery cannot see:
      - CA 2023 presence in firmware DB
      - DBX (deny list) size as rough revocation indicator
      - Boot manager signature on the EFI System Partition (the real one
        firmware loads, not the OS-side staging copy)

    Also reports OS-side file signatures and registry servicing state for
    side-by-side comparison.

    READ-ONLY. Makes no changes to the system.

.NOTES
    Intended use:
      - Confirm hosts reported as `compliant_via_registry` in the Fleet
        report are actually migrated at firmware level
      - Diagnose hosts stuck in unusual servicing states

    Exit code: always 0 unless PowerShell itself errors. Output is the deliverable.
#>

$ErrorActionPreference = 'Continue'

function Write-State {
    param([string]$Label, [string]$Value)
    Write-Output ("{0,-32} : {1}" -f $Label, $Value)
}

Write-Output "=== Windows Secure Boot CA 2023 verification ==="
Write-Output ""

# --- Secure Boot enabled ---
$sb = $null
try { $sb = Confirm-SecureBootUEFI } catch {}
Write-State "Secure Boot enabled" $sb

# --- Firmware DB: is CA 2023 trusted? ---
Write-Output ""
Write-Output "--- Firmware DB ---"
try {
    $dbBytes = (Get-SecureBootUEFI db).bytes
    $dbText  = [Text.Encoding]::ASCII.GetString($dbBytes)
    Write-State "CA 2023 present"  ($dbText -match 'Windows UEFI CA 2023')
    Write-State "PCA 2011 present" ($dbText -match 'Microsoft Windows Production PCA 2011')
    Write-State "DB size (bytes)"  $dbBytes.Length
} catch {
    Write-State "DB read" "FAILED: $_"
}

# --- DBX (deny list) size as revocation indicator ---
Write-Output ""
Write-Output "--- DBX (deny list) ---"
try {
    $dbxBytes = (Get-SecureBootUEFI dbx).bytes
    Write-State "DBX size (bytes)" $dbxBytes.Length
    Write-Output "    Pre-revocation DBX is typically ~16-32 KB."
    Write-Output "    Post-revocation DBX is much larger (often 100+ KB)."
} catch {
    Write-State "DBX read" "FAILED: $_"
}

# --- OS-side staging files ---
Write-Output ""
Write-Output "--- OS-side files (staging copies) ---"

$bootFiles = @(
    "$env:SystemRoot\Boot\EFI\bootmgfw.efi",
    "$env:SystemRoot\System32\winload.efi",
    "$env:SystemRoot\System32\winresume.efi"
)
foreach ($f in $bootFiles) {
    $name = Split-Path $f -Leaf
    if (-not (Test-Path $f)) {
        Write-State $name "MISSING"
        continue
    }
    $issuer = $null
    try { $issuer = (Get-AuthenticodeSignature $f).SignerCertificate.Issuer } catch {}
    $tag = if ($issuer -match 'Windows UEFI CA 2023')      { 'CA 2023' }
           elseif ($issuer -match 'Production PCA 2011')   { 'PCA 2011' }
           else { "unknown: $issuer" }
    Write-State $name $tag
}

# --- ESP boot manager (the one firmware actually loads) ---
Write-Output ""
Write-Output "--- ESP boot manager (the real one firmware loads) ---"

$mountLetter = "S"
$mounted     = $false
try {
    mountvol "${mountLetter}:" /s 2>&1 | Out-Null
    $mounted = ($LASTEXITCODE -eq 0)

    $espPath = "${mountLetter}:\EFI\Microsoft\Boot\bootmgfw.efi"
    if (Test-Path $espPath) {
        $espSig    = Get-AuthenticodeSignature $espPath
        $espIssuer = $espSig.SignerCertificate.Issuer
        $tag = if ($espIssuer -match 'Windows UEFI CA 2023')    { 'CA 2023 (firmware MIGRATED)' }
               elseif ($espIssuer -match 'Production PCA 2011') { 'PCA 2011 (firmware NOT migrated)' }
               else { "unknown: $espIssuer" }
        Write-State "ESP\EFI\...\bootmgfw.efi" $tag
        Write-State "Signature status" $espSig.Status
    } else {
        Write-State "ESP boot manager" "MISSING at $espPath"
    }
} catch {
    Write-State "ESP read" "FAILED: $_"
} finally {
    if ($mounted) { mountvol "${mountLetter}:" /d 2>&1 | Out-Null }
}

# --- Registry servicing state ---
Write-Output ""
Write-Output "--- Registry servicing state ---"

$servicing  = "HKLM:\SYSTEM\CurrentControlSet\Control\SecureBoot\Servicing"
$secureboot = "HKLM:\SYSTEM\CurrentControlSet\Control\Secureboot"

$status    = (Get-ItemProperty -Path $servicing  -Name "UEFICA2023Status"         -ErrorAction SilentlyContinue).UEFICA2023Status
$capable   = (Get-ItemProperty -Path $servicing  -Name "WindowsUEFICA2023Capable" -ErrorAction SilentlyContinue).WindowsUEFICA2023Capable
$err       = (Get-ItemProperty -Path $servicing  -Name "UEFICA2023Error"          -ErrorAction SilentlyContinue).UEFICA2023Error
$available = (Get-ItemProperty -Path $secureboot -Name "AvailableUpdates"         -ErrorAction SilentlyContinue).AvailableUpdates

$statusStr  = if ($status)              { $status }              else { "(not set)" }
$capableStr = if ($null -ne $capable)   { $capable.ToString() }  else { "(not set)" }
$errStr     = if ($null -ne $err)       { $err.ToString() }      else { "(not set)" }
$availStr   = if ($null -ne $available) { "0x{0:X4}" -f $available } else { "(not set)" }

Write-State "UEFICA2023Status"          $statusStr
Write-State "WindowsUEFICA2023Capable"  $capableStr
Write-State "UEFICA2023Error"           $errStr
Write-State "AvailableUpdates"          $availStr

Write-Output ""
Write-Output "Done. No changes made."
exit 0
