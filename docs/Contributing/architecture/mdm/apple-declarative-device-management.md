# Declarative Device Management (DDM) Architecture

This document provides an overview of how Fleet handles Apple’s Declarative Device Management (DDM) feature within its MDM architecture.

## Introduction

Declarative Device Management (DDM) is Apple’s [declarative paradigm extension][1] to the MDM protocol, designed to avoid the common performance and scalability issues associated with traditional MDM commands. With DDM, configuration settings are described declaratively, allowing devices to evaluate and enforce compliance without requiring continuous server polling.

## Architecture Overview

Fleet’s DDM architecture centers around the definition of DDM custom settings (JSON configuration profiles), their delivery to targeted hosts, and the state synchronization between the Fleet server and those Apple devices. The system ensures that each MDM-enrolled device receives the appropriate declarations and that the delivery and installation status is tracked over time.

## DDM lifecycle

The DDM custom settings are managed either via Fleet's UI, the API or `fleetctl gitops`. Then the Fleet server is responsible to deliver these settings to the targeted devices, and then to track the statuses. The following sub-sections expand on those lifecycle steps.

### Managing DDM custom settings

As for other types of custom settings, DDM profiles are associated with a team (or "No team") and can be applied to the team's hosts conditionally via labels.

Via the UI, all custom settings are set in the "Controls -> OS settings -> Custom settings" page. Apple's "traditional" configuration profiles (`.mobileconfig` files), Apple's DDM `.json` profiles and Windows' `.xml` profiles can all be uploaded and managed here.

Via the API (which is used by the UI), the following endpoints support DDM profiles:
* `GET /api/latest/fleet/configuration_profiles` lists all configuration profiles, including DDM profiles. See [the API reference](https://fleetdm.com/docs/rest-api/rest-api#list-custom-os-settings-configuration-profiles).
* `POST /api/latest/fleet/configuration_profiles` uploads a new configuration profile, which may be a DDM profile. See [the API reference](https://fleetdm.com/docs/rest-api/rest-api#add-custom-os-setting-configuration-profile).
* `DELETE /api/latest/fleet/configuration_profiles/{profile_uuid}` deletes a configuration profile, which may be a DDM profile. See [the API reference](https://fleetdm.com/docs/rest-api/rest-api#delete-custom-os-setting-configuration-profile).
* `GET /api/latest/fleet/configuration_profiles/summary` provides team-level statistics of custom settings by status, including DDM profiles. See [the API reference](https://fleetdm.com/docs/rest-api/rest-api#get-os-settings-summary).
* `GET /api/latest/fleet/configuration_profiles/{profile_uuid}` provides either the metadata or the profile's content (as a file attachment) for a specific profile, which may be a DDM one. See [the API reference](https://fleetdm.com/docs/rest-api/rest-api#get-or-download-custom-os-setting-configuration-profile).
* `GET /api/latest/fleet/configuration_profiles/{profile_uuid}/status` provides statistics of a specific custom settings by status, including DDM profiles. See [the API reference](https://fleetdm.com/docs/rest-api/rest-api#get-os-setting-configuration-profile-status).
* `POST /api/latest/fleet/hosts/{host_id}/configuration_profiles/{profile_uuid}/resend` resends a specific profile to a specific host, which may be a DDM profile (which is interesting, since the batch-resend doesn't support DDM). See [the API reference](https://fleetdm.com/docs/rest-api/rest-api#resend-custom-os-setting-configuration-profile).

Note that the following endpoints do _not_ support DDM profiles:
* `GET /api/latest/fleet/hosts/{id}/configuration_profiles` lists only the Apple `.mobileconfig` profiles of the host, not the DDM profiles nor the Windows profiles. See [the API reference](https://fleetdm.com/docs/rest-api/rest-api#get-configuration-profiles-assigned-to-a-host).
* `POST /api/_version_/fleet/configuration_profiles/resend/batch` batch-resends a specific configuration profile to all hosts where it is in a specific satus (e.g. "failed"). Does not support re-sending a DDM profile. See [the API reference](https://fleetdm.com/docs/rest-api/rest-api#batch-resend-custom-os-setting-configuration-profile).


Via `fleetctl gitops`, the following YAML section can be used to manage profiles:

```
controls:
  macos_settings:
    custom_settings:
      - path: ../lib/macos-profile1.mobileconfig
        labels_exclude_any:
          - Macs on Sequoia
      - path: ../lib/macos-profile2.json
        labels_include_all:
          - Macs on Sonoma
```

See full YAML reference [here](https://fleetdm.com/docs/configuration/yaml-files#macos-settings-and-windows-settings).

The `gitops` command uses the `POST /api/latest/fleet/mdm/profiles/batch` contributor-only API endpoint to set the profiles. It replaces any existing profile with the set provided in the YAML, removing any profile that is not present in the YAML.

### Delivery of DDM custom settings

### Verification of DDM custom settings

## Database details

The DDM profiles are stored in the `mdm_apple_declarations` table which closely resembles the `mdm_apple_configuration_profiles` table but uses `declaration_uuid` instead of `profile_uuid` as primary key. Note that for historical reasons, the `uuid` primary key column of both these tables and the `mdm_windows_configuration_profiles` table is `VARCHAR(37)` even though a UUID is 36 characters long. This is because a prefix is prepended to the generated UUID to distinguish the type of the profile, so that if you have its UUID, you know in which table to look for it. The [prefixes](https://github.com/fleetdm/fleet/blob/bd027dc4210b113983c3133251b51754e7d24c6f/server/fleet/mdm.go#L18-L20) are "d" for a DDM, "a" for an Apple `.mobileconfig` profile and "w" for a Windows profile.

The profiles names must be unique across all platforms and profile types for a given team (or "no team"), so [the SQL statement to insert new profiles is a bit unusual](https://github.com/fleetdm/fleet/blob/bd027dc4210b113983c3133251b51754e7d24c6f/server/datastore/mysql/apple_mdm.go#L5078-L5096). That's because distinct tables are used to store the different profile types, so a standard database constraint cannot be used.

## Supported features and limitations

* As mentioned earlier, label restrictions (include any, include all and exclude any) are supported for DDM profiles, same as for other types of profiles.
* Fleet secrets [are supported](https://github.com/fleetdm/fleet/blob/bd027dc4210b113983c3133251b51754e7d24c6f/server/service/apple_mdm.go#L885-L888) and are expanded with their values when the profile is created (or batch-set).
* Fleet _variables_ [are **not** supported](https://github.com/fleetdm/fleet/blob/bd027dc4210b113983c3133251b51754e7d24c6f/server/service/apple_mdm.go#L948-L953) for DDM.
* DDM profiles [cannot include OS updates settings](https://github.com/fleetdm/fleet/blob/bd027dc4210b113983c3133251b51754e7d24c6f/server/fleet/apple_mdm.go#L670-L672), as those are handled by Fleet via the "Controls -> OS updates" settings.
* DDM profiles [cannot be of a type that requires assets](https://github.com/fleetdm/fleet/blob/bd027dc4210b113983c3133251b51754e7d24c6f/server/fleet/apple_mdm.go#L674-L676), as assets are currently not supported.
* DDM profiles [cannot have a "status subscription" type](https://github.com/fleetdm/fleet/blob/bd027dc4210b113983c3133251b51754e7d24c6f/server/fleet/apple_mdm.go#L678-L680).
* DDM profiles [must be a configuration type](https://github.com/fleetdm/fleet/blob/bd027dc4210b113983c3133251b51754e7d24c6f/server/fleet/apple_mdm.go#L682-L684).

## Architecture diagram

```
[Placeholder for Automated Device Enrollment Architecture Diagram]
```

## Special Cases

- **Device Re-enrollment**: On device re-enrollment, Fleet re-applies all relevant declarations.
- **Manual Removal or Tampering**: If a declaration is removed or tampered with on the device, Fleet detects this via status updates and re-issues as needed.
- **Setup assistant**:

## Related Resources

- [Original research on DDM](https://docs.google.com/document/d/1FRpIdIShpM4nEhPI5FH0Arqg-NO_e-nBMqXJWjJRnSs/edit?tab=t.0)
- [MDM Product Group Documentation](../../product-groups/mdm/) - Documentation for the MDM product group
- [MDM Development Guides](../../guides/mdm/) - Guides for MDM development

[1]: https://developer.apple.com/documentation/devicemanagement/leveraging-the-declarative-management-data-model-to-scale-devices
