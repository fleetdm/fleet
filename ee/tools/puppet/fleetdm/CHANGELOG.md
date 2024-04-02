# Changelog

All notable changes to this project will be documented in this file.

## Release 0.2.4

**Bug Fixes**

- If a profile preassignment fails during the run, the profile matcher won't update the profiles in the Fleet server.
- Improved error handling for different API calls during profile preassignment to avoid crashing the Puppet run if some of them fail.

## Release 0.0.0-beta.1

**Features**

- Ability to define profiles using the custom type `fleetdm::profile`.
- Ability to release a device from await configuration using the custom function `fleetdm::release_device`.

