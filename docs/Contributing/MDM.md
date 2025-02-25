# MDM index

## Knowledge transfer videos

- Roberto's MDM knowledge transfer videos
    - [Google Drive](https://drive.google.com/drive/u/0/folders/1xvR89NNJWEc2dG0Se9m0goTz0Pjh1-RN)
    - Also on Loom -- search for `MDM knowledge dump`

- Marcos's Windows knowledge transfer videos
    - [Session1](https://drive.google.com/file/d/1d4rcK2bsLGVocbh2s88vW2FNzOQxP1B_)
    - [Session2](https://drive.google.com/file/d/1V5Jl7azXnZZRnkjwDaEvF1pe24nVZHSH)

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

    (Note: Despite its name, this cron job handles both Apple and Windows MDM. [Issue link](https://github.com/fleetdm/fleet/issues/22824))

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
    1. The proxy for the seamless flow lives in `./tools/mdm/migration/mdmproxy/`
    2. The tool to extract data from MicroMDM lives in `./tools/mdm/migration/micromdm/touchless/`

### Disk encryption

Enabling and retrieving the disk encryption keys for a host behaves differently depending on the OS.

After the key is retrieved, it's stored in the `host_disk_encryption_keys` table. The value for the key is encrypted using Fleet's CA certificate, and thus can only be decrypted if you have the CA private key.

**FileVault (macOS)**

For macOS, disk encryption involves a two step process:

1. Sending a profile with two payloads:
    1. A Payload to configure how the disk is going to be encrypted
    2. A Payload to configure the escrow of the encryption key

2. Retrieving the disk encryption key:
    1. Via osquery, we grab the (encrypted) disk encryption key
    2. In a cron job, we verify that we're able to decrypt the key. It's necessary to verify if a key is encrypted because we could have grabbed a key generated by a third-party MDM, or an invalid key.

If we're not able to decrypt the key for a host, the key needs to be rotated. Rotation happens silently by:

1. The server sends a notification to orbit, notifying that the key couldn't be decrypted.
2. orbit installs an authorization plugin named [Escrow Buddy](https://github.com/macadmins/escrow-buddy) that performs the key rotation the next time the user logs in.
3. Fleet retrieves and tries to validate the key again.

**BitLocker (Windows)**

Disk encryption in Windows is performed entirely by orbit.

When disk encryption is enabled, the server sends a notification to orbit, who calls the [Win32_EncryptableVolume class](https://learn.microsoft.com/en-us/windows/win32/secprov/getencryptionmethod-win32-encryptablevolume) to encrypt/decrypt the disk and generate an encryption key.

After the disk is encrypted, orbit sends the key back to the server using an orbit-authenticated endpoint (`POST /api/fleet/orbit/disk_encryption_key`)

### Load testing

osquery-perf supports MDM load testing for Windows and Apple devices. Under the hood it uses the `mdmtest` package to simulate MDM clients.

Documentation about setting up load testing for MDM can be found in ./infrastructure/loadtesting/terraform/readme.md

### ADE

For a high-level overview of how ADE works please check the [glossary](https://github.com/fleetdm/fleet/blob/main/tools/mdm/apple/glossary-and-protocols.md
)

Below is a summary of Fleet-specific behaviors for ADE.

### Sync

Sincronization of devices from all ABM tokens uploaded to Fleet happen in the `dep_syncer` cron job, which runs every 30 seconds.

We keep a record of all devices ingested via the ADE sync in the `host_dep_assignments` table. Entries in this table are soft-deleted.

On every run, we pull the list of added/modified/deleted devices and:

1. If the host was added/modified, we:
    1. Create/match a row in the `hosts` table for the new host. This allows IT admin to move the host between teams before it turns on MDM or has `fleetd` installed.
    1. Assign the corresponding JSON profile to each host using ABM's APIs.
2. If the host was deleted, we soft delete the `host_dep_assignments` entry 

#### Special case: host in ABM is deleted in Fleet

If an IT admin deletes a host in the UI/API, and we have a non-deleted entry in `host_dep_assignments` for the host, we immediately create a new host entry as if the device was just ingested from the ABM sync.

### IdP integration

If the IT admin configured an MDM IdP integration, we change the `configuration_web_url` value in the JSON profile to be `{server_url}/mdm/sso`, this page initiates the SSO flow in the setup assistant webview.

Key points about he IdP flow:

1. The SSO flow ends with a callback to the Fleet server which contains information about the user that just logged in. We store this information in the `mdm_idp_accounts` table. Because at this point we don't know from which host UUID the request is coming in, we generate a random UUID as the key to look up this information.
2. The Fleet server responds with an enrollment profile, that contains a special `ServerURL` with a query parameter `enrollment_reference`, this parameter has the random UUID generated in step 1.
3. During MDM enrollment, we grab the `enrollment_reference` parameter, if present, and we try to match it to a host. This allows us to link end user IdP accounts used during enrollment with a host.
4. Before releasing the device from awaiting configuration, we send an `AccountConfiguration` command to the host, to pre-set the macOS local account user name to the value we got stored in `mdm_idp_accounts`

