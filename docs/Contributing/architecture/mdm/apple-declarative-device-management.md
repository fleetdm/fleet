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

TODO: tables, uuid prefix, etc.

### Delivery of DDM custom settings

### Verification of DDM custom settings

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
