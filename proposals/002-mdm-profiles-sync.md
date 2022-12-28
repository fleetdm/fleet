# MDM Configuration Profiles

## Summary

Fleet MDM will need to store and sync configuration profiles to MDM enrolled devices.
Configuration profiles are sometimes referred to as “MacOS settings” in the Fleet UI.
Whenever there is a change to configuration profiles or a device enrolls, configuration profiles will need to be synced to the device.

## Storage

- Store the profiles in a `mdm_apple_configuration_profiles` table with the following columns (incomplete)
    - `payload_identifier`: this should be unique, but stay the same when editing an existing profile. Use something like `com.fleetdm.<random uuid>`.
    Built-in profiles that are included with fleet should use a hard coded identifier so that we can update these profiles.
    This may be required when new options are added or bug fixes are required to an existing profile
    - `payload_uuid`: this should change whenever an existing profile is updated
        - note, this doesn't actually need to be a uuid
    - `version`: isn't part of the actually configuration profile. Used to determine what the latest version of a profile is. For simplicity, this can be an integer that is incremented from 1.
        - note, this is *not* the payload_version which is used by apple to track the format version of the profile, usually `1`.
    - `raw`: The complete raw configuration profile as xml (mobileconfig)
    - example query to get the latest version of all the profiles
        ```
        SELECT p1.*
        FROM
            mdm_apple_configuration_profiles p1
            LEFT JOIN mdm_apple_configuration_profiles p2
                  ON p1.payload_identifier = p2.payload_identifier AND p1.version < p2.version
        WHERE
            p2.version is NULL
        ```
    - How do we handle deleted/disabled profiles? What about builtin profiles?
        - soft delete ie deleted_at. Need to clean these up periodically?
        - delete the profile, and allow `host_mdm_apple_configuration_profiles.payload_identifier` to be orphaned

- Store the current profiles installed on mdm enrolled hosts in a `host_mdm_apple_configuration_profiles` table
    - `host_id`, `payload_identifier`, `payload_uuid`
    - Populate whenever a `InstallProfile` or periodic `ProfileList` MDM command is run on a host

## Sync

- backend terminology
    - current state: The profiles and their specific versions that are installed on devices
    - desired state: The latest profiles and their specific versions
    - drift: Differnce between the current state and the desired state
    - refresh: detect drift and reconcile by installing/removing profiles
- frontent terminology
    - applied: host has all profiles with the latest version installed.
    - pending: drift has been detected on the host eg a new profile was created in fleet which needs to be installed on the host
    - failed: there was an error with the profile itself, or installing the profile on the host

- To avoid issues when syncing profiles eg side effects, we should only install/remove profiles that have actually changed
    - detect drift/out
- Add a mechanism to refresh the list of profiles installed on an individual host manually and sync the desired profiles, similar to the osquery refetch host mechanism
    - helps in troubleshooting
    - able to quickly test changes ie during development
- To avoid unexpected behaviour and to increase robustness, periodically issue a `ProfileList` MDM command on hosts
- If a profile is added and removed from fleet quickly, how should we handle `InstallProfile` commands that are in the queue, but haven't been executed on hosts yet.

- How should we handle profiles that have been removed from fleet, but haven't been removed from hosts yet
