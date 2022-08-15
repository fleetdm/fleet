# Fleet Sandbox & Pre-Packaged Fleet-osquery Installers

## Goals

1. Improve UX on Fleet Sandbox by offering pre-packaged Fleet-osquery installers.
2. Add the "Pre-Packaged installers" feature to "Fleet Sandbox" as soon as possible (i.e. not block on having a fully functional "Fleet Packager" service).

## Fleet Sandbox Assumptions

- We will limit number of teams to T.
- Sandbox has good root CA trusted certificates
- Users won't be allowed to change enroll secrets.

## Pre-Packaged Installers Plan

We will need some changes to fleetctl, the pre-provisioner, the Fleet server and UI.

### fleetctl

Make all functionality in `fleetctl package` to run in linux. (This change will be needed for the Packager service anyways.)

PS: Abstract in its own package so that it can be used by Packager service in a next iteration.

### Pre-provisioner

Following are the pre-provisioner steps to generate the pre-packaged installers:

1. Generate T+1 random enroll secrets.
	
2. Run `fleetctl package --type={pkg|rpm|deb|msi}` with T+1 enroll secrets (i.e. one for Global and one for each team).
PS: There's some complexity in storing/handling credentials for macOS Signing and Notarization of the packages.

3. The generated packages will be stored in a S3 bucket accessible by the Fleet server with the following object name format
`$INSTALLERS_DIR/$ENROLL_SECRET/fleet-osquery.$TYPE`, e.g. `/fleet-installers/FzRCZWTlEY2kqzIwk1BE9fru5KuhrlYP/fleet-osquery.pkg`.
We propose using S3 to support multiple Fleet instances serving the requests.

4. Set comma-separated `FLEET_ENROLL_POOL` environment variable to Fleet server config (Fleet would use those secrets instead of randomly generating one).
The Fleet server will only serve the installers with enroll secret listed in this variable (security check).

### Fleet Server and UI

- Fleet server new configuration and new functionality:
	- `FLEET_MAX_TEAMS`: Maximum number of teams to allow in the deployment (default 0, disabled).
	- `FLEET_DISABLE_ENROLL_CHANGE`: Disallow users from changing enroll secrets (default false).
	- `FLEET_PACKAGES_S3_*`: S3 configuration for the retrieval of the pre-packaged installers (default empty).
	- `FLEET_ENROLL_POOL`: comma-separated enrolls to use when needed (default empty), must have equals to FLEET_MAX_TEAMS+1 items (default empty).

- Fleet will serve a new authenticated API (for Sandbox-only): `{GET|HEAD} /api/v1/fleet/download_installer/{enroll}/{type}`, e.g. `GET /api/v1/fleet/download_installer/FzRCZWTlEY2kqzIwk1BE9fru5KuhrlYP/rpm`.
    - The UI can make a `HEAD` request to check if an installer exists, if so, then it can display a download button for it, (if not, "show the current UI"? TBD with UI team)
    - The API looks for the installer corresponding to the Global/Team the user is looking at, and returns it for download.
