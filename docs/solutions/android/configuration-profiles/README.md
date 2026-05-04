# Android Configuration Profiles


## [Password Policy](password-policy.json)

- This disables pattern and swipe, by only allowing PIN and password locks.
- Google limits how the complexity requirements function on BYOD devices. For BYOD:
  - `passwordMinimumLength` set to 8 makes the length requirement for PIN 8, password 6.
  - `passwordMinimumLength` set to 6 makes the length requirement for PIN 6, password 4.
