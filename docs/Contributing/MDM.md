# MDM index

## Protocol

### Apple

- Apple documentation for MDM
    - Web version: https://developer.apple.com/documentation/devicemanagement
    - YAML version: https://github.com/apple/device-management
        - NOTE: documentation for upcoming/beta features is available in the YAML version first, look for branches.

- Fleet's MDM glossary: https://github.com/fleetdm/fleet/blob/main/tools/mdm/apple/glossary-and-protocols.md

- MicroMDM and NanoMDM wikis.

- MacAdmins Slack, channels: #mdmdev, #declarative-management, #apple-feedback, #nudge, #swiftdialog

### Windows

- Enrollment protocol: https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mde/5c841535-042e-489e-913c-9d783d741267

- MDM protocol: https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mdm/33769a92-ac31-47ef-ae7b-dc8501f7104f

- CSPs: https://learn.microsoft.com/en-us/windows/client-management/mdm/policy-configuration-service-provider

### Development

- `./Testing-and-local-development.md` has many sections about setting up MDM and debugging specific features
- `tools/mdm/apple/troubleshooting.md` has instructions to troubleshoot Apple MDM in hosts
- `tools/mdm/*` contains many ad-hoc tools we've been adding to aid development, including but not limited to:
    - Servers to perform MDM migrations from different providers
    - Tools to debug the ABM APIs
    - Tools to import/export certificates and keys from the DB

## Fleet specific features

### Profiles and Declarations

For Windows and Apple MDM, Fleet defines "profiles" as groups of settings that can be SyncML (Windows), XML, or JSON (Apple).

Settings are defined per team and can be further targeted to hosts using labels.

Profiles and declarations can be uploaded via the UI/CLI by the IT admin. Additionally, Fleet automatically sends profiles to hosts as part of high-level UI actions. For instance, enabling disk encryption in macOS sends the relevant configuration to the host.

To determine the subset of profiles/declarations that should be applied to a specific host, we use the following approach:

1. **Ideal State:** The "ideal state" of a host is calculated by combining team profiles with any label-based inclusions/exclusions using the `mdm_*_configuration_profiles` and `mdm_apple_declarations` tables.
2. **Current State:** The "current state" of a host is tracked using the `host_mdm_*_configuration_profiles` and `host_mdm_apple_declarations` tables, which are updated based on MDM protocol responses and kept in sync via osquery (commonly referred to as "profile verification" or "double-check").
3. **Diff Calculation:** We use set algebra to compute the difference between the ideal and current states—determining the profiles that need to be installed ("to install") and those that should be removed ("to remove").

This logic runs in two main places:

1. **`mdm_apple_profile_manager` Cron Job:** Runs every 30 seconds and performs the following actions:
    - Calculates the profiles to install/remove
    - Enqueues the necessary commands
    - Sends push notifications to the hosts

    (Note: Despite its name, this cron job handles both Apple and Windows MDM.)

2. **`ds.BulkSetPendingMDMHostProfiles`:** This method is called to mark profiles and declarations as "pending." It's a lighter process than the cron job, only updating database records to reflect pending profiles in the UI, providing immediate feedback to users.

### Profile/declaration verification (double check)

Osquery double checks are explained in good detail as part of their initial specs:

- https://github.com/fleetdm/fleet/issues/11099
- https://github.com/fleetdm/fleet/issues/9780

### NanoMDM Integration for Apple MDM

[NanoMDM](https://github.com/micromdm/nanomdm) handles the core protocol operations for Apple MDM. Fleet extends the protocol and adds custom handling to align with our desired workflows.

After NanoMDM processes a protocol operation, it allows for custom logic by implementing the `CheckinAndCommandService` interface. Our implementation can be found in the service layer.

https://github.com/fleetdm/fleet/blob/4ff5f7a18abcaa9003fe250fb50d7a246d2af795/server/service/apple_mdm.go#L2595-L2600

Additionally, per request from management, we maintain a modified version of the NanoMDM repository under `server/mdm/nanomdm/` to support Fleet-specific requirements.

### MDM Lifecycle

While implementing MDM-specific features, we identified recurring patterns related to managing the "MDM connection" to the host. We refer to this as the "MDM lifecycle." A few examples:

- The MDM server is not always notified when a host unenrolls. As a result, we perform cleanup during host enrollment since it's impossible to reliably determine whether a host is enrolling for the first time.
- Special MDM actions are triggered when a host is deleted via the UI.

All lifecycle-related actions are implemented in the `HostLifecycle` struct. A great way to familiarize yourself with the different MDM actions is to search for the usages of the different `HostAction`s we have.

We use the terminology "turn on" and "turn off" to align with the language used by the product group, which is generally associated with enrollment and unenrollment.

### SCEP Renewals

MicroMDM has documented the intricacies of SCEP certificate renewals in detail: [MicroMDM SCEP Documentation](https://github.com/micromdm/micromdm/wiki/Device-Identity-Certificate-Expiration).

In Fleet, we issue certificates with a 1-year validity period. To renew certificates, we send a new enrollment profile via the `InstallProfile` command.

The command to install the profile is queued in the `renew_scep_certificates` job, which is part of the `cleanups_then_aggregation` cron schedule.

It's important to note that when sending a new enrollment profile, certain fields must remain unchanged. Apple has documented these restrictions here: [Apple MDM Profile Restrictions](https://github.com/apple/device-management/blob/85fae8ac896578447f8fcb07ff6c976128133a9c/mdm/profiles/com.apple.mdm.yaml#L1).

### Puppet module

The implementation of the Puppet module is described in detail at: `ee/tools/puppet/fleetdm/CONTRIBUTING.md`

### MDM migrations

Windows MDM is more flexible when it comes to switching MDM servers, so MDM migrations are generally not a big deal.

Apple MDM is more strict, and we have built two different flows:

1. The "regular" flow is what most customers will use, involve `fleetd` guiding the user through the migration to perform manual steps. The user documentation for this flow is https://fleetdm.com/guides/mdm-migration
2. The "seamless" flow allows customers with access to their MDM database and ownership of the domain used as the `ServerURL` in the enrollment profile to migrate the devices without user action. The user documentation for this flow is https://fleetdm.com/guides/seamless-mdm-migration

### Disk encryption

**FileVault**

**BitLocker**

### ABM/ADE

- ABM integration and custom behaviors
- Automatic enrollment additions

