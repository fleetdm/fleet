# windows-migrate-to-fleet-mdm.ps1
#
# Moves a Windows device from its current MDM (such as Intune) to Fleet MDM,
# with no user interaction. Add it to Fleet as a saved script and run it on one
# host or a batch. Before you use it, set $FleetServerUrl below.
#
# Steps:
#   1. Preflight checks. Nothing changes if any check fails.
#   2. Unenroll from the current MDM. The Entra join stays and the device
#      isn't reset.
#   3. Ask Fleet to refetch the host, and wait until Fleet shows MDM Off.
#   4. Enroll in Fleet MDM, then confirm the device has checked in with Fleet.
#
# Requirements: Windows 10/11 (not Server), fleetd installed with scripts
# enabled and connected to Fleet, Windows MDM turned on in Fleet, and a Fleet
# script timeout of at least 300 seconds (the default).
#
# Safe to rerun. The orbit node key and device token are read locally, sent
# only to Fleet, and never printed.
#
# Exit codes:
#   0  Enrolled in Fleet and checked in. Nothing to do.
#   1  Preflight failed (not SYSTEM, 32-bit, Windows Server, fleetd missing,
#      URL not set, or enrolled in a different Fleet server). No changes.
#   2  Something on the device would re-enroll it in the old MDM
#      (AutoEnrollMDM Group Policy or ConfigMgr client). No changes.
#   3  Fleet unreachable, device token rejected, or Windows MDM off. No changes.
#   4  Unenroll failed. Still on the old MDM.
#   5  An enrollment Fleet counts as MDM On is still in the registry. See output.
#   6  No MDM: Fleet didn't show MDM Off in time. Rerun.
#   7  No MDM: Fleet enrollment failed. Rerun, then see output.
#   8  Enrolled in Fleet, but not checked in yet. Rerun later to confirm.

$FleetServerUrl = 'https://fleet.example.com'
# Set to $true only if ConfigMgr co-management won't re-enroll the device in Intune.
$AllowConfigMgrClient = $false

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

$deadline = (Get-Date).AddSeconds(280)   # stay under Fleet's 300-second script timeout
$script:unmanaged = $false               # true once the device has no MDM

function Log([string]$msg) { Write-Output ('[{0:HH:mm:ss}] {1}' -f (Get-Date), $msg) }

function Finish([int]$code, [string]$msg) {
  Log "RESULT exit=$code : $msg"
  exit $code
}

function Get-SecondsLeft { [int][Math]::Max(0, ($deadline - (Get-Date)).TotalSeconds) }

# Flags for one HKLM\SOFTWARE\Microsoft\Enrollments key:
#   Active      - a real MDM enrollment: EnrollmentState 1, or an OMADM account
#                 exists. Some working enrollments report EnrollmentState 3.
#   FleetSeesOn - Fleet reports MDM On for this key (UPN value present, state
#                 0-3 or missing, discovery URL set) and won't accept enrollment.
function Get-EnrollmentFlags($props, [bool]$hasOmadm) {
  $builtInProviders = @('Deploy Authority', 'WMI_Bridge_SCCM_Server', 'Cloud Authority', 'Local Authority')
  $state = $null
  if ($null -ne $props.EnrollmentState) { $state = [int]$props.EnrollmentState }
  $isMdmProvider = [bool]($props.ProviderID -and $builtInProviders -notcontains $props.ProviderID)
  [pscustomobject]@{
    State       = $state
    Active      = ($isMdmProvider -and ($state -eq 1 -or $hasOmadm))
    FleetSeesOn = (($props.PSObject.Properties.Name -contains 'UPN') -and ($null -eq $state -or $state -in 0..3) -and [string]$props.DiscoveryServiceFullURL -ne '')
  }
}

function Get-Enrollments {
  Get-ChildItem 'HKLM:\SOFTWARE\Microsoft\Enrollments' -ErrorAction SilentlyContinue | ForEach-Object {
    $p = Get-ItemProperty -LiteralPath $_.PSPath -ErrorAction SilentlyContinue
    if (-not $p) { return }
    $hasOmadm = Test-Path -LiteralPath "HKLM:\SOFTWARE\Microsoft\Provisioning\OMADM\Accounts\$($_.PSChildName)"
    $f = Get-EnrollmentFlags $p $hasOmadm
    [pscustomobject]@{
      Id          = $_.PSChildName
      ProviderID  = [string]$p.ProviderID
      State       = $f.State
      OMADM       = $hasOmadm
      Active      = $f.Active
      FleetSeesOn = $f.FleetSeesOn
    }
  }
}

function Show-Enrollments([string]$title) {
  $rows = @(Get-Enrollments | Where-Object { $_.Active -or $_.FleetSeesOn })
  if ($rows.Count -eq 0) { Log "$title : none"; return }
  Log "$title :"
  $rows | Select-Object Id, ProviderID, State, OMADM, Active, FleetSeesOn |
    Format-Table -AutoSize | Out-String -Width 200 | Write-Output
}

function Test-FleetEnrolled { [bool](Get-Enrollments | Where-Object { $_.Active -and $_.ProviderID -eq 'Fleet' }) }

# Reads the Fleet enrollment's OMADM account. Returns None, WrongServer,
# Pending (no successful check-in yet), or Synced (LastSessionResult 0).
function Get-FleetEnrollmentHealth {
  $e = Get-Enrollments | Where-Object { $_.Active -and $_.ProviderID -eq 'Fleet' } | Select-Object -First 1
  if (-not $e) { return 'None' }
  $acct = "HKLM:\SOFTWARE\Microsoft\Provisioning\OMADM\Accounts\$($e.Id)\Protected"
  $addr = [string](Get-ItemProperty -LiteralPath "$acct\AddrInfo" -ErrorAction SilentlyContinue).Addr
  if ($addr -and $addr -notmatch [Regex]::Escape(([Uri]$FleetServerUrl).Host)) { return 'WrongServer' }
  $conn = Get-ItemProperty -LiteralPath "$acct\ConnInfo" -ErrorAction SilentlyContinue
  if ($conn -and $null -ne $conn.LastSessionResult -and [int]$conn.LastSessionResult -eq 0) { return 'Synced' }
  return 'Pending'
}

function Wait-FleetSynced([int]$seconds) {
  $limit = (Get-Date).AddSeconds([Math]::Min($seconds, (Get-SecondsLeft)))
  do {
    $h = Get-FleetEnrollmentHealth
    if ($h -eq 'Synced' -or $h -eq 'WrongServer') { return $h }
    Start-Sleep -Seconds 5
  } while ((Get-Date) -lt $limit)
  return (Get-FleetEnrollmentHealth)
}

function Finish-FleetEnrolled([string]$how) {
  $h = Wait-FleetSynced 90
  Show-Enrollments 'MDM enrollments after'
  [void](Request-Refetch)
  switch ($h) {
    'Synced'      { Finish 0 "$how First management session with Fleet succeeded." }
    'WrongServer' { Finish 1 "Enrolled in a Fleet server other than $FleetServerUrl. Check `$FleetServerUrl." }
    default       { Finish 8 "$how No successful management session with Fleet yet. Rerun later to confirm." }
  }
}

function Test-EntraJoined { [bool]((dsregcmd /status 2>$null) | Select-String -Pattern '^\s*AzureAdJoined\s*:\s*YES') }

# Returns status and body for error responses too. Never logs the URL, which
# can contain the device token.
function Invoke-Http([string]$Method, [string]$Url, [string]$Body = $null, [string]$ContentType = 'application/json') {
  $resp = $null
  try {
    $req = [System.Net.HttpWebRequest]::Create($Url)
    $req.Method = $Method
    $req.Timeout = 20000
    $req.ReadWriteTimeout = 20000   # Timeout doesn't cover reading the body
    $req.UserAgent = 'fleet-windows-mdm-migration'
    if ($Body) {
      $bytes = [Text.Encoding]::UTF8.GetBytes($Body)
      $req.ContentType = $ContentType
      $req.ContentLength = $bytes.Length
      $s = $req.GetRequestStream(); $s.Write($bytes, 0, $bytes.Length); $s.Close()
    } elseif ($Method -ne 'GET') {
      $req.ContentType = $ContentType
      $req.ContentLength = 0
    }
    $resp = $req.GetResponse()
  } catch {
    $ex = $_.Exception
    while ($ex -and -not ($ex -is [System.Net.WebException])) { $ex = $ex.InnerException }
    if (-not $ex -or -not $ex.Response) {
      $m = if ($ex) { $ex.Message } else { $_.Exception.Message }
      return [pscustomobject]@{ Status = 0; Body = $m }
    }
    $resp = $ex.Response
  }
  try {
    $reader = New-Object System.IO.StreamReader($resp.GetResponseStream())
    $text = $reader.ReadToEnd(); $reader.Close()
    return [pscustomobject]@{ Status = [int]$resp.StatusCode; Body = $text }
  } finally { $resp.Close() }
}

# fleetd rotates this token hourly, so read it fresh each time.
function Get-DeviceToken {
  if (-not (Test-Path $deviceTokenPath)) { return $null }
  $t = (Get-Content -Path $deviceTokenPath -Raw).Trim()
  if ($t) { return $t } else { return $null }
}

# Fleet's record of this host, from the same device API that Fleet Desktop uses.
function Get-FleetHostView {
  $tok = Get-DeviceToken
  if (-not $tok) { return [pscustomobject]@{ Ok = $false; Status = 0; Error = 'device token file missing or empty' } }
  $r = Invoke-Http GET "$fleetBase/api/latest/fleet/device/$tok`?exclude_software=true"
  if ($r.Status -ne 200) { return [pscustomobject]@{ Ok = $false; Status = $r.Status; Error = $r.Body } }
  $h = ($r.Body | ConvertFrom-Json).host
  $status = $null; $name = $null
  if ($h.mdm) { $status = $h.mdm.enrollment_status; $name = $h.mdm.name }
  [pscustomobject]@{ Ok = $true; Status = 200; RefetchRequested = [bool]$h.refetch_requested; MdmStatus = $status; MdmName = $name }
}

function Request-Refetch {
  $tok = Get-DeviceToken
  if (-not $tok) { return $false }
  $r = Invoke-Http POST "$fleetBase/api/latest/fleet/device/$tok/refetch"
  return ($r.Status -ge 200 -and $r.Status -lt 300)
}

function Fail-Unmanaged([int]$code, [string]$msg) {
  Show-Enrollments 'MDM enrollments now'
  Finish $code "$msg Device is NOT managed by any MDM. Rerun this script."
}

try {
  # --- 1. Preflight ---
  Log 'Step 1/4: preflight (no changes)'

  if ([Environment]::Is64BitOperatingSystem -and -not [Environment]::Is64BitProcess) {
    Finish 1 'Running in 32-bit PowerShell; registry and DLL calls would hit the WOW64 views. Run in 64-bit PowerShell (Fleet does by default).'
  }
  if (-not [Security.Principal.WindowsIdentity]::GetCurrent().IsSystem) {
    Finish 1 'Must run as SYSTEM. Run it from Fleet (Host > Run script).'
  }
  $installType = (Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion' -ErrorAction SilentlyContinue).InstallationType
  if ($installType -match 'Server') {   # 'Server' and 'Server Core'
    Finish 1 'Windows Server. Fleet MDM does not enroll Windows Server hosts.'
  }
  if (-not $FleetServerUrl -or $FleetServerUrl -match 'example\.com') {
    Finish 1 'Set $FleetServerUrl at the top of the script to your Fleet server URL.'
  }
  $fleetBase = $FleetServerUrl.TrimEnd('/')
  $discoveryUrl = "$fleetBase/api/mdm/microsoft/discovery"

  $orbitDir = Join-Path $env:ProgramFiles 'Orbit'
  $nodeKeyPath = Join-Path $orbitDir 'secret-orbit-node-key.txt'
  $deviceTokenPath = Join-Path $orbitDir 'identifier'

  if (Test-FleetEnrolled) {
    Finish-FleetEnrolled 'Already enrolled in Fleet MDM; no changes made.'
  }

  if (-not (Test-Path $nodeKeyPath)) { Finish 1 "Orbit node key not found at $nodeKeyPath. Is fleetd installed?" }
  $nodeKey = (Get-Content -Path $nodeKeyPath -Raw).Trim()
  if (-not $nodeKey) { Finish 1 "Orbit node key at $nodeKeyPath is empty." }
  if (-not (Get-DeviceToken)) { Finish 1 "fleetd device token not found at $deviceTokenPath. Is fleetd installed and running?" }

  $view = Get-FleetHostView
  if (-not $view.Ok) {
    Finish 3 "Fleet device API check failed (HTTP $($view.Status)). Check that $fleetBase is reachable from the host and is the URL fleetd uses. Detail: $($view.Error)"
  }
  Log "Fleet currently reports MDM: $($view.MdmStatus) ($($view.MdmName))"

  # An empty POST returns "Windows MDM isn't turned on" when MDM is off, and a
  # SOAP fault when it's on.
  $disco = Invoke-Http POST $discoveryUrl '' 'application/soap+xml; charset=utf-8'
  if ($disco.Status -eq 0) { Finish 3 "Can't reach Fleet's Windows MDM discovery endpoint: $($disco.Body)" }
  if ($disco.Status -eq 404) { Finish 3 'Fleet returned 404 for the Windows MDM discovery endpoint. Check $FleetServerUrl.' }
  if ($disco.Body -match "Windows MDM isn.t turned on") { Finish 3 'Windows MDM is turned off in Fleet. Turn it on in Settings > Integrations > MDM before migrating.' }

  $mdmPolicy = Get-ItemProperty 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\CurrentVersion\MDM' -ErrorAction SilentlyContinue
  if ($mdmPolicy -and $mdmPolicy.AutoEnrollMDM -eq 1) {
    Finish 2 'Group Policy "Enable automatic MDM enrollment using default Azure AD credentials" is set (AutoEnrollMDM=1). It would re-enroll the device into the old MDM. Remove the GPO from this device first.'
  }
  if (-not $AllowConfigMgrClient -and (Get-Service -Name CcmExec -ErrorAction SilentlyContinue)) {
    Finish 2 'ConfigMgr client (CcmExec) is installed. Co-management can re-enroll the device into Intune. Remove the client or co-management first, or set $AllowConfigMgrClient = $true if you have confirmed it will not.'
  }

  $entraBefore = Test-EntraJoined
  Log "Entra joined: $entraBefore"

  Add-Type -Namespace Win32 -Name MdmRegistration -MemberDefinition @'
[DllImport("mdmregistration.dll")] public static extern int UnregisterDeviceWithManagement(IntPtr enrollmentId);
[DllImport("mdmregistration.dll", CharSet = CharSet.Unicode)] public static extern int RegisterDeviceWithManagement(string upn, string discoveryUri, string accessToken);
'@

  # --- 2. Unenroll from the current MDM ---
  Show-Enrollments 'MDM enrollments before'
  $active = @(Get-Enrollments | Where-Object { $_.Active })

  if ($active.Count -eq 0) {
    Log 'Step 2/4: no active MDM enrollment; skipping unenroll.'
    $script:unmanaged = $true
  } else {
    Log ("Step 2/4: unenrolling from {0}" -f (($active | ForEach-Object { $_.ProviderID }) -join ', '))
    $hr = [Win32.MdmRegistration]::UnregisterDeviceWithManagement([IntPtr]::Zero)
    Log ('UnregisterDeviceWithManagement returned 0x{0:X8}' -f $hr)
    if ($hr -ne 0) { Finish 4 'Unenroll call failed. The device is still on its old MDM.' }
    $script:unmanaged = $true

    $limit = (Get-Date).AddSeconds([Math]::Min(90, (Get-SecondsLeft)))
    do {
      Start-Sleep -Seconds 5
      $left = @(Get-Enrollments | Where-Object { $_.Active -or $_.FleetSeesOn })
    } while ($left.Count -gt 0 -and (Get-Date) -lt $limit)

    if ($left.Count -gt 0) {
      Show-Enrollments 'MDM enrollments after unenroll'
      Finish 5 'An enrollment is still registered after unenroll; Fleet would keep reporting MDM On. Wait a few minutes and rerun. The old MDM may already be gone, so treat the device as possibly unmanaged.'
    }
    Log 'Unenrolled from old MDM.'

    $entraAfter = Test-EntraJoined
    if ($entraBefore -and -not $entraAfter) {
      Log 'WARNING: Entra join was present before unenroll and is not reported now. Check dsregcmd /status on this host.'
    }
  }

  # A stale key that Fleet counts as MDM On blocks enrollment even when no
  # MDM is active.
  if (@(Get-Enrollments | Where-Object { $_.FleetSeesOn }).Count -gt 0) {
    Show-Enrollments 'MDM enrollments'
    Finish 5 'Registry holds an enrollment Fleet reports as MDM On (stale or failed key with a discovery URL). Fleet will refuse enrollment until it is gone. Investigate before deleting anything.'
  }

  # --- 3. Wait for Fleet to show MDM Off ---
  # Fleet only accepts the enrollment once its record of the host shows no MDM.
  Log 'Step 3/4: asking Fleet to refetch and waiting for MDM Off'
  if (-not (Request-Refetch)) { Log 'Refetch request was not accepted; will keep polling.' }
  $refetchedAt = Get-Date
  $eligible = $false
  $limit = (Get-Date).AddSeconds([Math]::Max(0, (Get-SecondsLeft) - 90))
  while ((Get-Date) -lt $limit) {
    Start-Sleep -Seconds 15
    if (Test-FleetEnrolled) { Log 'fleetd enrolled the host on its own.'; break }
    $view = Get-FleetHostView
    if (-not $view.Ok) { Log "Fleet device API returned HTTP $($view.Status); retrying."; continue }
    if (-not $view.RefetchRequested -and ($null -eq $view.MdmStatus -or $view.MdmStatus -eq 'Off')) {
      Log 'Fleet reports MDM Off.'
      $eligible = $true
      break
    }
    if (((Get-Date) - $refetchedAt).TotalSeconds -ge 60) {
      [void](Request-Refetch); $refetchedAt = Get-Date
    }
  }
  if (-not $eligible -and -not (Test-FleetEnrolled)) {
    Fail-Unmanaged 6 'Fleet did not report MDM Off in time.'
  }

  # --- 4. Enroll in Fleet ---
  if (-not (Test-FleetEnrolled)) {
    Log 'Step 4/4: enrolling in Fleet MDM'
    $token = [ordered]@{ type = 1; payload = [ordered]@{ orbit_node_key = $nodeKey; auth_token = '' } } | ConvertTo-Json -Compress -Depth 3
    $hr = [Win32.MdmRegistration]::RegisterDeviceWithManagement('', $discoveryUrl, $token)
    # Logged only. A non-zero result can still leave a working enrollment (or
    # fleetd may enroll at the same moment), so the registry check decides.
    Log ('RegisterDeviceWithManagement returned 0x{0:X8}' -f $hr)

    $limit = (Get-Date).AddSeconds([Math]::Min(45, (Get-SecondsLeft)))
    while (-not (Test-FleetEnrolled) -and (Get-Date) -lt $limit) { Start-Sleep -Seconds 5 }
    if (-not (Test-FleetEnrolled)) {
      Fail-Unmanaged 7 ('No Fleet enrollment after RegisterDeviceWithManagement (0x{0:X8}). 0x8019xxxx codes are HTTP errors from Fleet (e.g. 0x80190190 = 400).' -f $hr)
    }
    Finish-FleetEnrolled 'Enrolled in Fleet MDM.'
  }
  Log 'Step 4/4: fleetd already enrolled the host; skipping enroll call.'
  Finish-FleetEnrolled 'Enrolled in Fleet MDM by fleetd.'
} catch {
  $msg = "Unexpected error: $($_.Exception.Message)"
  if ($script:unmanaged) { Fail-Unmanaged 7 $msg }
  Finish 1 "$msg No changes made."
}
