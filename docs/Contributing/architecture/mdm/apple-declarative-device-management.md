# Declarative Device Management (DDM) Architecture

This document provides an overview of how Fleet handles Apple’s Declarative Device Management (DDM) feature within its MDM architecture.

## Introduction

Declarative Device Management (DDM) is Apple’s [declarative paradigm extension](https://developer.apple.com/documentation/devicemanagement/leveraging-the-declarative-management-data-model-to-scale-devices) to the MDM protocol, designed to avoid the common performance and scalability issues associated with traditional MDM commands. With DDM, configuration settings are described declaratively, allowing devices to evaluate and enforce compliance without requiring continuous server polling.

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

Delivery of the DDM profiles is handled via a cron job, as for other types of profiles. The `ReconcileAppleDeclarations` function takes care of marking as "pending" the hosts that have declarations that changed (either to install or to remove). However, unlike the `ReconcileAppleProfiles` function that delivers non-declarative `.mobileconfig` profiles, the declarations job only needs to enqueue a [`DeclarativeManagement` MDM command](https://developer.apple.com/documentation/devicemanagement/declarativemanagementcommand) next, targeting all hosts with changed DDM profiles, and the DDM protocol will take care of the rest.

The reason we even need a `ReconcileAppleDeclarations` function is so that we can transition the statuses of the profiles on the host to "pending", and to ensure we initiate the DDM protocol via the `DeclarativeManagement` command (this is how the DDM protocol is handled - similar to, say, Websockets that are initiated via a standard HTTP request, the [DDM protocol is built on top of the traditional MDM protocol](https://developer.apple.com/documentation/devicemanagement/integrating-declarative-management) and requires that `DeclarativeManagement` command to get started).

Otherwise, the fact that we initiate it only for hosts that have changed DDM profiles is an optimization, sending it for the other hosts would simply detect that no changes were necessary.

There is one more thing happening in `ReconcileAppleDeclarations`, and it's to handle the `resync` field of the `host_mdm_apple_declarations` table. [This was added](https://github.com/fleetdm/fleet/pull/29059) to handle a race where a [declaration is pending install but is currently marked as pending remove](https://github.com/fleetdm/fleet/blob/afc37124eedde3a226137cca613adf3a0ff799c7/server/datastore/mysql/apple_mdm.go#L5590-L5595). In this case, the install is immediately transitioned to "verified" (as it had to be installed in order to be pending remove), but because the remove might've already happened (and Fleet did not get the status confirmation), the profile is marked as "to resync" for the host to ensure it gets delivered.

### Verification of DDM custom settings

The DDM protocol is handled by the `nanomdm` package, which calls out to a Fleet-implemented `DeclarativeManagement` interface:
* the `nanomdm` handler is here: https://github.com/fleetdm/fleet/blob/afc37124eedde3a226137cca613adf3a0ff799c7/server/mdm/nanomdm/service/nanomdm/service.go#L198
* the Fleet implementation of the `DeclarativeManagement` interface is here: https://github.com/fleetdm/fleet/blob/afc37124eedde3a226137cca613adf3a0ff799c7/server/service/apple_mdm.go#L5818
* the registration of the Fleet implementation in the `nanomdm` service handler is here: https://github.com/fleetdm/fleet/blob/afc37124eedde3a226137cca613adf3a0ff799c7/server/service/handler.go#L1229-L1230

When a DDM session is initiated (via the `DeclarativeManagement` command), `nanomdm` will call the `DeclarativeManagement` registered interface (in our case, the Fleet implementation) to do the actual exchange of DDM messages. The DDM protocol is based on a [series of messages executing different operations identified by the `Endpoint` field](https://developer.apple.com/documentation/devicemanagement/declarativemanagementrequest).

The [various endpoint operations are handled in the Fleet implementation](https://github.com/fleetdm/fleet/blob/afc37124eedde3a226137cca613adf3a0ff799c7/server/service/apple_mdm.go#L5833-L5852) by dispatching to different functions that return the requested information to the device. It also stores all messages received in the DDM protocol into the `mdm_apple_declarative_requests` table. Here's a breakdown of what each operation does:

* `Endpoint == "tokens"`: generates the token for the set of declarations to be sent to the host (a global token for the full set of declarations to apply). How that token is generated is [somewhat involved](https://github.com/fleetdm/fleet/blob/afc37124eedde3a226137cca613adf3a0ff799c7/server/datastore/mysql/apple_mdm.go#L5322-L5340). The generated token dictates if the host will receive the declarations or not, depending on the token of the last applied changes on the host.
* `Endpoint == "declaration-items"`: sends the list of declarations to install to the host. Only the (individual) tokens of the declarations are sent in this step (along with their activation token). Declarations to remove have their status transition from `nil` to "pending" as part of this processing (because since they are not included in the list sent to the host, the host will remove any declaration not in the set - this is how a "remove" is done with DDM). Every configuration needs an "activation" to be applied, so this also creates the corresponding activations. The host then determines which declaration is missing using the individual tokens, and requests the full declaration content as needed in a subsequent step.
* `strings.HasPrefix(Endpoint, "declaration/configuration")`: sends the full JSON of the corresponding declaration (identified by the "Endpoint"), expanding its Fleet secrets as needed.
* `strings.HasPrefix(Endpoint, "declaration/activation")`: sends the full JSON of the corresponding activation (identified by the "Endpoint"). Activations can be used to conditionally apply configurations, but we currently don't use that feature.
* `Endpoint == "status"`: receives the status report of the DDM profiles on the host. If the declaration is active and valid, it is marked as "verified", and if it is invalid it is marked as "failed". Other rare cases are handled in this code, but those are the main ones. Note that [according the Roberto's research at the time](https://github.com/fleetdm/fleet/blob/afc37124eedde3a226137cca613adf3a0ff799c7/server/service/apple_mdm.go#L6084-L6093), the host will not send "remove" statuses, instead we detect removal by the fact that the declaration is not in the status report.

In addition to verifying the DDM profiles from the status response of the DDM protocol, we also [update the statuses from the response of the traditional `DeclarativeManagement` command](https://github.com/fleetdm/fleet/blob/afc37124eedde3a226137cca613adf3a0ff799c7/server/service/apple_mdm.go#L3486) to do the initial transition from "pending" to "verifying" or "failed" depending on the result of the command. This batch-affects all declarations for the host.

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

- **Setup assistant**: As of Fleet v4.70.0, DDM profiles are ignored during the setup assistant phase of a macOS device. This is because Apple does not currently support the DDM protocol in this phase, it always returns a "NotNow" response to the `DeclarativeManagement` MDM command, so Fleet cannot wait for DDM profiles to be delivered before releasing the device so we ignore them and wait for after the device is configured to send them.

## Related Resources

- [Original research on DDM](https://docs.google.com/document/d/1FRpIdIShpM4nEhPI5FH0Arqg-NO_e-nBMqXJWjJRnSs/edit?tab=t.0)
- [MDM Product Group Documentation](../../product-groups/mdm/) - Documentation for the MDM product group
- [MDM Development Guides](../../guides/mdm/) - Guides for MDM development
