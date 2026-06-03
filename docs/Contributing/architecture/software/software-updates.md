# Software updates architecture

This document provides an overview of Fleet's software updates architecture.

## Introduction

Software updates in Fleet enables the management and deployment of software updates across the device fleet. This document provides insights into the design decisions, system components, and interactions specific to the Software updates functionality.

## Architecture overview

The software updates architecture enables the identification, configuration, and deployment of software updates across a fleet of devices. It leverages platform-specific mechanisms to update software on devices.

## Key components

## Architecture diagram

```
[Placeholder for software updates architecture diagram]
```

## Update flow

### Sourcing the latest version

Fleet sources up-to-date versions for software in different ways for software installers (FMAs) and VPP apps.

- **VPP:** The `refresh_vpp_app_versions` cron job runs hourly, and fetches the latest versions for every available app. This updates their rows in the `vpp_apps` table. Android apps are made available for the host to install through the Google Play Store, and Fleet doesn't handle any installs/updates for them other than during setup experience.
- **Fleet-maintained apps:** On every gitops run, Fleet gets the latest manifest for every Fleet-maintained app, and downloads the latest version if there is a new one. It also caches the previous version for the rollback feature. 
- **Custom installers:** Can be edited manually to use a newer version of the software.  

### Triggering updates

There are multiple ways for Fleet to trigger an actual install of the latest version on a host:

**VPP auto-updates:**
This is only available for iOS and iPadOS hosts. When an iOS/iPadOS host returns the results of the `InstalledApplicationList` command as part of a refetch, Fleet will check which apps need to be updated based on the available inventory, and sends install commands for apps that are outdated. Fleet will only do this if the host's local time is within the install window for that software title in the `software_update_schedules` table.

**Policies:**
Fleet can trigger software installs, both VPP and  FMA/custom installers, through the software automation available for policies. When a host fails a policy with a software install automation, Fleet calls either `InsertSoftwareInstallRequest` or `InsertHostVPPSoftwareInstall` which will install the latest software on the failing host. Note that any policy can have a software automation, and it is only tied to whether the policy fails or passes.

There are two types of automatic policies that can be generated:
- **Automatic install policies:** these are generated only at install time since they pull from the `exists` query in the FMA manifest. The policy checks whether the app is installed on the host at all, and has a software automation to install that app.
- **Patch policies:** these can be generated at any point for an FMA, they pull from the `patched` query in the FMA manifest but are kept in the `software_installers` row for the installer. They are kept up to date whenever a new version of the FMA is added to Fleet. By default, they do not include the software install automation, so they don't trigger any installs, but can be modified to do so. Currently, patch policies are implemented to fail if any outdated versions of the software exist on the host, and pass if either none at all or only the current version is installed. 

## Related resources

- [Software product group documentation](../../product-groups/software/) - Documentation for the software product group
