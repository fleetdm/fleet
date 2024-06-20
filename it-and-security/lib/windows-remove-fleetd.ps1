function Test-Administrator
{
    [OutputType([bool])]
    param()
    process {
        [Security.Principal.WindowsPrincipal]$user = [Security.Principal.WindowsIdentity]::GetCurrent();
        return $user.IsInRole([Security.Principal.WindowsBuiltinRole]::Administrator);
    }
}

# borrowed from Jeffrey Snover http://blogs.msdn.com/powershell/archive/2006/12/07/resolve-error.aspx
function Resolve-Error-Detailed($ErrorRecord = $Error[0]) {
  $error_message = "========== ErrorRecord:{0}ErrorRecord.InvocationInfo:{1}Exception:{2}"
  $formatted_errorRecord = $ErrorRecord | format-list * -force | out-string
  $formatted_invocationInfo = $ErrorRecord.InvocationInfo | format-list * -force | out-string
  $formatted_exception = ""
  $Exception = $ErrorRecord.Exception
  for ($i = 0; $Exception; $i++, ($Exception = $Exception.InnerException)) {
    $formatted_exception += ("$i" * 70) + "-----"
    $formatted_exception += $Exception | format-list * -force | out-string
    $formatted_exception += "-----"
  }

  return $error_message -f $formatted_errorRecord, $formatted_invocationInfo, $formatted_exception
}

#Stops Orbit service and related processes
function Stop-Orbit {

  # Stop Service
  Stop-Service -Name "Fleet osquery" -ErrorAction "Continue"
  Start-Sleep -Milliseconds 1000

  # Ensure that no process left running
  Get-Process -Name "orbit" -ErrorAction "SilentlyContinue" | Stop-Process -Force
  Get-Process -Name "osqueryd" -ErrorAction "SilentlyContinue" | Stop-Process -Force
  Get-Process -Name "fleet-desktop" -ErrorAction "SilentlyContinue" | Stop-Process -Force
  Start-Sleep -Milliseconds 1000
}

#Remove Orbit footprint from registry and disk
function Force-Remove-Orbit {

  try {

    #Stoping Orbit
    Stop-Orbit

    #Remove Service
    $service = Get-WmiObject -Class Win32_Service -Filter "Name='Fleet osquery'"
    if ($service) {
      $service.delete() | Out-Null
    }

    #Removing Program files entries
    $targetPath = $Env:Programfiles + "\\Orbit"
    Remove-Item -LiteralPath $targetPath -Force -Recurse -ErrorAction "Continue"

    #Remove HKLM registry entries
    Get-ChildItem "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall" -Recurse  -ErrorAction "SilentlyContinue" |  Where-Object {($_.ValueCount -gt 0)} | ForEach-Object {

      # Filter for osquery entries
      $properties = Get-ItemProperty $_.PSPath  -ErrorAction "SilentlyContinue" |  Where-Object {($_.DisplayName -eq "Fleet osquery")}
      if ($properties) {

        #Remove Registry Entries
        $regKey = "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\" + $_.PSChildName

        Get-Item $regKey -ErrorAction "SilentlyContinue" | Remove-Item -Force -ErrorAction "SilentlyContinue"

        return
      }
    }
  }
  catch {
    Write-Host "There was a problem running Force-Remove-Orbit"
    Write-Host "$(Resolve-Error-Detailed)"
    return $false
  }

  return $true
}

function Main {

  try {
    # Is Administrator check
    if (-not (Test-Administrator)) {
      Write-Host "Please run this script with adming privileges."
      Exit -1
    }

    Write-Host "About to uninstall fleetd..."

    if ($mode -eq "remove") {
        # "remove" is received as argument to the script when called as the
        # sub-process that will actually remove the fleet agent.

        # sleep to give time to fleetd to send the script results to Fleet
        Start-Sleep -Seconds 20
        if (Force-Remove-Orbit) {
            Write-Host "fleetd was uninstalled."
            Exit 0
        } else {
            Write-Host "There was a problem uninstalling fleetd."
            Exit -1
        }
    } else {
        # when this script is executed from fleetd, it does not immediately
        # remove the agent. Instead, it starts a new detached process that
        # will do the actual removal. This is done to avoid the agent being
        # killed by the removal process, which prevents it from sending the
        # script execution results to Fleet, causing the script to remain
        # "Pending" and being re-executed when/if the host reinstalls the
        # agent.
        #
        # See https://github.com/fleetdm/fleet/issues/19197#issuecomment-2150020270
        $execName = $MyInvocation.ScriptName
        $proc = Start-Process -PassThru -FilePath "powershell" -WindowStyle Hidden -ArgumentList "-MTA", "-ExecutionPolicy", "Bypass", "-File", "$execName remove"
        Start-Sleep -Seconds 5 # give time to process to start running
        Write-Host "Removal process started: $($proc.Id)."
    }

  } catch {
    Write-Host "Errorr: Entry point"
    Write-Host "$(Resolve-Error-Detailed)"
    Exit -1
  }
}

$mode = $args[0]
$null = Main
