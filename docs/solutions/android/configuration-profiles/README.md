# Android Configuration Profiles


## [Password Policy](password-policy.json)

- This disables pattern and swipe, by only allowing PIN and password locks.
- Google limits how the complexity requirements function on BYOD devices. For BYOD:
  - `passwordMinimumLength` set to 8 makes the length requirement for PIN 8, password 6.
  - `passwordMinimumLength` set to 6 makes the length requirement for PIN 6, password 4.


## [Disable face and biometrics unlock](disable-face-and-biometrics-unlock.json)

- Android applies these values to the work profile lock. On BYOD hosts, the end user has one lock for both the work profile and the host by default. There is no separate work profile lock to restrict, so Android restricts the host's lock instead. This turns off fingerprint and face unlock for the whole host, including the end user's personal apps.
- To restrict biometric unlock on the work profile only, add the settings from [Require separate work profile lock](require-separate-work-profile-lock.json) to this profile.


## [Require separate work profile lock](require-separate-work-profile-lock.json)

- This stops the end user from using one lock for both the work profile and the host.
- Until the end user sets a work profile lock, Android reports `passwordPolicies` with a reason of `USER_ACTION`, and Fleet shows the profile as "Failed" on **Host > OS settings**. The profile moves to "Verified" after the end user sets the lock.
- `unifiedLockSettings` requires Android 9 or later, and Android rejects the policy unless `passwordScope` is `SCOPE_PROFILE`.
