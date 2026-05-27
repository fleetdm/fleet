# Setup Experience Overview

Setup experience lets newly enrolled hosts be configured with all the MDM profiles, software, and scripts that they would need. And optionally block them from completing setup until requirements like all required software being installed are met.

## Summary

Setup experience has various triggers depending on the platform and MDM enrollment. After enrollment, orbit goes through end user authentication, then calls the `/api/fleet/orbit/setup_experience/init` endpoint which causes the Fleet server to queue up the relevant items in the `setup_experience_status_results` table with pending statuses. For ADE enrollments, Fleet enqueues the items when handling the `TokenUpdate` request. 

The orbit `SetupExperiencer` config receiver runs every 30s and requests the current setup experience status. This status lets orbit know if config profiles, software installers and scripts are still pending or finished while those happen asynchronously through MDM and unified queue activities. Once all items are completed, the end user can exit setup experience.

## Enqueuing items

Fleet enqueues relevant software installers, VPP apps, and scripts during ADE enrollment or after orbit calls `setup_experience/init`. This is what the end user sees on the web UI. When Fleet processes a `TokenUpdate` request with `AwaitingConfiguration==true` for ADE enrollments, it enqueues MDM items (profiles, bootstrap package, account configurations) as MDM commands in the nanomdm command queue.

## Setup experience status

Whenever `/api/fleet/orbit/setup_experience/status` is called, the Fleet server checks the current state of MDM profiles, software installs, and script runs. At the end of the function, it will call `SetupExperienceNextStep()` which queues up the next item (software install/VPP install/script run) for the host in the unified queue (see [upcoming activities](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/guides/upcoming-activities.md)), and updates the status in `setup_experience_status_results` for that item. If there is nothing left to do, the device is released. On macOS, if everything is done and the device hasn't been released manually, the server sends a `DeviceConfigured` MDM command here to release the device from Setup Assistant.

These activities run in order, but they don't directly interact with the setup experience which is why the setup experience receiver polls for status. The results from software installs, script runs, or VPP installs are responsible for updating the setup experience status for that item. See [software installation](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/architecture/software/software-installation.md) for how Fleet installs software on hosts while setup experience is running.

## Platform differences

### Windows

Windows enrollment goes through the Enrollment Status Page (ESP) rather than Apple's Setup Assistant, and Fleet does not push profiles, a bootstrap package, or account configuration as part of setup experience on Windows. Setup experience is driven by orbit, and the device is held in OOBE (Out Of Box Experience) until orbit calls `/api/fleet/orbit/setup_experience/init`. Only software installers are supported on Windows, scripts are macOS-only.

### Linux

Linux has no MDM, so there is no MDM command path at all. Setup experience starts when orbit calls `/api/fleet/orbit/setup_experience/init` on boot, and the same 30s polling loop drives software installs (no scripts).

### iOS/iPadOS

iOS and iPadOS don't run orbit, so there is no polling loop and no `init` endpoint. Fleet only enqueues VPP apps for setup experience items on the first `TokenUpdate`. The release worker on the Fleet server polls internally until every item is done before sending `DeviceConfigured` to exit Setup Assistant.

### Android

Setup experience is not supported on Android. Apps are delivered through the device policy at enrollment time, and there is no step-by-step flow for the end user.


## Flow diagram (orbit)

```mermaid
flowchart TB

    subgraph FS["Fleet Server"]
        ESE["EnqueueSetupExperienceItems()"]
        TU["TokenUpdate handler\n(ADE: AwaitingConfiguration==true)"]
        TU --> ESE

        subgraph GOSES["GetOrbitSetupExperienceStatus()"]
            direction TB
            G_MDM["bootstrap package status\nconfiguration profile statuses\naccount configuration status\n(macOS only)"]
            G_SW["get software/script statuses\nfrom setup_experience_status_results"]
            G_FI{"failed installs?"}
            G_RFSS["ResetSetupExperienceItemsAfterFailure()\nif resetFailedSetupSteps == true and requireAllSoftware == true"]
            G_RCSA["recordCanceledSetupExperienceSoftwareActivities()"]
            G_SENS["SetupExperienceNextStep()\nenqueues next software / VPP / script install"]
            G_REL{"ready for release?"}
            G_DEVCFG["DeviceConfigured MDM command\n(macOS: exit Setup Assistant)"]

            G_MDM --> G_SW --> G_FI
            G_FI -->|"yes"| G_RFSS --> G_RCSA
            G_FI -->|"no"| G_RCSA
            G_RCSA --> G_REL
            G_REL -->|"yes"| G_DEVCFG
            G_REL -->|"no"| G_SENS
        end

        UQ[/"unified queue · upcoming_activities"/]
        G_SENS --> UQ

        SIR_EP(("POST /orbit/software_install/result\nSaveHostSoftwareInstallResult()"))
        SCR_EP(("POST /orbit/scripts/result\nSaveHostScriptResult()"))

        SIR_EP -.->|"maybeUpdateSetupExperienceStatus\nupdates setup_experience_status_results"| G_SW
        SCR_EP -.->|"maybeUpdateSetupExperienceStatus\nupdates setup_experience_status_results"| G_SW
    end

    subgraph OH["Orbit Host"]
        direction LR
        O_START(["processSetupExperience"])
        O_CONN{"server.Has(\nCapabilityWebSetupExperience\n)?"}
        O_EUA{"end user auth"}
        O_INIT["orbit host calls init endpoint\nInitiateSetupExperience()"]
        O_SECR(("SetupExperience config receiver\nNewSetupExperiencer()"))
        O_CSAS["call setupExperienceStatus()"]
        O_PP{"profiles/bootstrap/\naccount config pending?"}
        O_RUI["render web UI"]
        O_SP{"any software\ninstalls pending?"}
        O_SCP{"any scripts\npending?"}
        O_RETURN["receiver returns"]
        O_DONE(["done — show close button"])
        O_SWRX(("software install receiver"))
        O_SCRX(("script receiver"))

        O_START --> O_CONN
        O_CONN -->|"yes"| O_EUA
        O_EUA -->|"fail"| O_EUA
        O_EUA -->|"success"| O_INIT
        O_INIT --> O_SECR
        O_SECR --> O_CSAS
        O_CSAS --> O_PP
        O_PP -->|"yes"| O_RETURN
        O_PP -->|"no"| O_RUI --> O_SP
        O_SP -->|"yes"| O_RETURN
        O_SP -->|"no"| O_SCP
        O_SCP -->|"yes"| O_RETURN
        O_SCP -->|"no"| O_DONE
        O_RETURN -. "next run in 30s" .-> O_SECR
    end

    O_INIT -->|"calls"| ESE
    ESE -.->|"Enabled=true → register receiver"| O_SECR
    O_CSAS -->|"calls"| GOSES

    UQ -.->|"delivered to orbit"| O_SWRX
    UQ -.->|"delivered to orbit"| O_SCRX
    O_SWRX -->|"calls"| SIR_EP
    O_SCRX -->|"calls"| SCR_EP
```

## Related resources

- [Upcoming activities](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/guides/upcoming-activities.md) - How Fleet's unified activity queue works
- [Software installation architecture](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/architecture/software/software-installation.md) - How Fleet installs software on hosts
- [Automated Device Enrollment](automated-device-enrollment.md) - Architecture for ADE
