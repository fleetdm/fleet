# Microsoft Teams MSIX Detection Fix

## Problem Summary

Microsoft Teams failed GitHub Actions validation because osquery couldn't detect the installed MSIX package:
```
level=error ts=2025-11-20T19:27:21.3052927Z caller=main.go:176 app="Microsoft Teams"
  msg="App version '25306.804.4102.7193' was not found by osquery"
```

## Root Cause

The installation script used only `Add-AppProvisionedPackage`:
```powershell
Add-AppProvisionedPackage -Online -PackagePath $msixPath -SkipLicense
```

**Problem**: `Add-AppProvisionedPackage` provisions the package for all *future* users but does NOT install it for the current user. The package only appears in a user's registry (where osquery looks) after that user logs in.

From Microsoft documentation:
> "Provisioned apps cannot be pre-installed; they are installed when a user account is created, and provisioning adds the files needed to install, but on its own does not install."

## How osquery Detects MSIX Packages

osquery's MSIX detection (added in PR #8585) queries per-user registry locations:
```
HKEY_USERS\<SID>\Software\Classes\Local Settings\Software\Microsoft\Windows\CurrentVersion\AppModel\Repository\Packages
```

If the package isn't registered in the current user's registry, osquery cannot detect it.

## Investigation Results

Testing on Windows VM confirmed:
- osquery 5.18.1 CAN detect MSIX packages ✓
- osquery CAN detect Teams when properly installed ✓
- Standalone `osqueryi --json` works correctly ✓
- 133 MSIX packages detected successfully ✓

The issue was NOT osquery's MSIX support (which works perfectly), but the installation method.

## Solution

Use BOTH commands to:
1. Provision for all future users
2. Install for current user immediately (so osquery can detect it)

**Updated Install Script** (`msteams_install.ps1`):
```powershell
try {
    # Provision for all future users
    Add-AppProvisionedPackage -Online -PackagePath $msixPath -SkipLicense

    # Also install for current user so osquery can detect it immediately
    Add-AppxPackage -Path $msixPath
} catch {
    Write-Host "Error: $_.Exception.Message"
    $exitCode = 1
}
```

**Updated Uninstall Script** (`msteams_uninstall.ps1`):
```powershell
try {
    # Remove for current user
    Remove-AppxPackage -Package (Get-AppxPackage -Name $packageName).PackageFullName

    # Also remove provisioned package for all future users
    Get-AppxProvisionedPackage -Online | Where-Object { $_.DisplayName -eq $packageName } | Remove-AppxProvisionedPackage -Online
} catch {
    Write-Host "Error: $_.Exception.Message"
    $exitCode = 1
}
```

## Why This Works

- `Add-AppxPackage` installs the package for the current user's registry → osquery can detect it immediately
- `Add-AppProvisionedPackage` provisions for future users → any new user that logs in will get Teams automatically
- Both commands together ensure Teams is available now AND for future users

## Testing Notes

After this fix, the validation workflow should:
1. Download Teams MSIX
2. Run install script (provisions AND installs for current user)
3. Run osquery validation
4. osquery finds Teams in current user's registry ✓
5. Validation passes ✓

## Files Changed

- `ee/maintained-apps/inputs/winget/scripts/msteams_install.ps1` - Added `Add-AppxPackage`
- `ee/maintained-apps/inputs/winget/scripts/msteams_uninstall.ps1` - Added provisioned package removal

## References

- osquery MSIX support PR: https://github.com/osquery/osquery/pull/8585
- Microsoft docs on provisioned vs installed packages
- Diagnostic testing results showing osquery successfully detecting Teams
