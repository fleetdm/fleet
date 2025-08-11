# Android configuration profiles design

Fleet uses configuration profile interface to apply configuration to hosts. This interface is cross-platform and will be used for Android as well.

Android configuration profiles are JSON files that are created with settings available in [policy](https://developers.google.com/android/management/reference/rest/v1/enterprises.policies) resource in [Android Management API](https://developers.google.com/android/management/reference/rest) (AMAPI).

## Andorid Management API (AMAPI) structure

- Enterprise: represents single organization. Fleet creates enterprise for each customer who enables Android MDM under Fleet's Google Cloud project.
- Devices: devices are associated to an enterprise. Each host that is enrolled for one organization will be associated with enterprise.
- Policies: configurations that are created inside the enterprise. Device can have only one policy associated. Every setting from the policy resource can be applied to a host.

## Android policy and Fleet configuration profile

In Fleet policy resource can be split into multiple files that can be scoped with labels.

### Example profiles:

`android-cross-profile-config.json`

```json
{
 "crossProfilePolicies": {
   "crossProfileCopyPaste": "COPY_FROM_WORK_TO_PERSONAL_DISALLOWED"
 },
}
```

`android-restrictions.json`

```json
{
 "screencaptureDisabled": true,
 "cameraDisabled": true
}
```

`android-password-requirements.json`

```json
{
 "passwordRequirements": {
    "passwordQuality": "COMPLEX",
   "passwordMinimumLength": 12,
   "passwordMinimumNonLetter": 1 ,
   "passwordMinimumUpperCase": 1,
   "passwordScope": "SCOPE_DEVICE"
 },
}
```

If for the reference, all these profiles are in the same team (no labels applied), hosts will apply a policy that have all settings from all profiles. 

In case of duplicate settings, Fleet will apply one that's delivered to a host most recently.

## Mapping configuration profiles to policies in AMAPI

For each host that enrolls Fleet will create policy and assign it. Probably we could take `enrollment_id` and use it as policy ID.

Let's say this is enrollment ID: `B54Z-7OY3-NHH7-FASO6-MSPN-O24B-2`

Policy will have this ID: `enterprises/{enterpriseID}/policies/B54Z-7OY3-NHH7-FASO6-MSPN-O24B-2`

Response from [enterprises.device.get](https://developers.google.com/android/management/reference/rest/v1/enterprises.devices/get):

```json
{
  "name": "enterprises/{enterpriseID}/devices/333b5673b7649c99",
  "managementMode": "PROFILE_OWNER",
  "state": "ACTIVE",
  "appliedState": "ACTIVE",
  "policyCompliant": true,
  "enrollmentTime": "2025-08-05T12:09:50.719Z",
  "lastStatusReportTime": "2025-08-11T12:17:30.534Z",
  "lastPolicySyncTime": "2025-08-11T18:29:05.611Z",
  "appliedPolicyVersion": "1",
  "apiLevel": 34,
  "hardwareInfo": {
    "brand": "samsung",
    "hardware": "mt6768",
    "deviceBasebandVersion": "A055FXXS9CYE1,A055FXXS9CYE1",
    "manufacturer": "samsung",
    "serialNumber": "B54Z-7OY3-NHH7-FASO6-MSPN-O24B-2",
    "model": "SM-A055F",
    "enterpriseSpecificId": "B54Z-7OY3-NHH7-FASO6-MSPN-O24B-2"
  },
  "policyName": "enterprises/{enterpriseId}/policies/B54Z-7OY3-NHH7-FASO6-MSPN-O24B-2",
  "appliedPolicyName": "enterprises/{enterpriseId}/policies/B54Z-7OY3-NHH7-FASO6-MSPN-O24B-2",
  "memoryInfo": {
    "totalRam": "3857195008",
    "totalInternalStorage": "4594905088"
  },
  "userName": "enterprises/{enterpriseId}/users/115581330686717215569",
  "enrollmentTokenName": "enterprises/{enterpriseId}/enrollmentTokens/IxBF09OYzTtktq5-Q99TuMa6yg100M_hbTgtCMomXYY",
  "previousDeviceNames": [
    "enterprises/{enterpriseId}/devices/33ec715d08b615d2"
  ],
  "securityPosture": {
    "devicePosture": "SECURE"
  },
  "ownership": "PERSONALLY_OWNED"
}
```

You can find policy that's assigned to a host under `appliedPolicyName`. There's also `policyName` which is used to show if other policy is associated to a host but not yet applied (host is offline). In Fleet's case we won't need `policyName` as it will be always associated with same policy.

We should use `appliedPolicyVersion` to know if new settings (e.g. new configuration profile uploaded) are applied.

Response from [enterprises.policies.get](https://developers.google.com/android/management/reference/rest/v1/enterprises.policies/get):

```json
{
  "name": "enterprises/{enterpriseId}/policies/B54Z-7OY3-NHH7-FASO6-MSPN-O24B-2",
  "version": 1,
  "crossProfilePolicies": {
   "crossProfileCopyPaste": "COPY_FROM_WORK_TO_PERSONAL_DISALLOWED"
  },
  "screencaptureDisabled": true,
  "cameraDisabled": true,
  "passwordRequirements": {
    "passwordQuality": "COMPLEX",
   "passwordMinimumLength": 12,
   "passwordMinimumNonLetter": 1 ,
   "passwordMinimumUpperCase": 1,
   "passwordScope": "SCOPE_DEVICE"
  },
}
```

### Updating policy

Fleet should patch policy each time when new setting is added to Fleet through configuration profiles, or when scope changes (labels).

[enterprises.policies.patch](https://developers.google.com/android/management/reference/rest/v1/enterprises.policies/patch) supports `updateMask` field, that can be used to indicate which field to update. If a new profile appeared in the device scope, Fleet will update the existing policy with the new fields.

### Default policy

Fleet should include `statusReportingSettings` in a policy for each host, so Fleet can collect more host vitals that aren't available without this policy. We [already do this](https://github.com/fleetdm/fleet/blob/18ff73007d17af1563b15cff0bcdc81a56d48819/server/mdm/android/service/service.go#L271), so we should match it.


## Verification of configuration profiles

...