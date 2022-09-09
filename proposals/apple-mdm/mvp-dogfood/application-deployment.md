# Application deployment

## MVP-Dogfood deliverable

- Application management via `fleetctl` commands.
- No teams support. Configured apps will be installed globally ("global team").
- No Munki, apps are deployed via the `InstallEnterpriseApplication` MDM command.

## Backend design

New MySQL table `macos_installers` that represents Fleet-hosted installers:
- `id`
- `name VARCHAR(255)` (UNIQUE): Name of the installer item.
- `sha256 VARCHAR(64)` (UNIQUE): SHA-256 hex-encoded of the installer item.
- `manifest TEXT` plist for MDM deployment
- `installer LARGEBLOB` // for MVP-Dogfood only, to not depend on S3 storage
- `url_token VARCHAR(36)` // UUID generated randomly

The column url_token is needed because installing applications via the [InstallEnterpriseApplication command](https://developer.apple.com/documentation/devicemanagement/installenterpriseapplicationcommand/command) does not support authentication on the package URL specified in the manifest.

## Fleetctl commands

`fleetctl apple-mdm installers upload --path=some-app.pkg`
1. Uploads installer to Fleet.
2. Creates an entry in `macos_installers`.
3. Outputs <INSTALLER_ID>.

`fleetctl apple-mdm installers list`
Output installer `macos_installers` entries, including `manifest` to stdout.

`fleetctl apple-mdm installers delete --id=<INSTALLER_ID>`
Remove entry from `macos_installers`

`fleetctl apple-mdm installers install-via-mdm --id=<INSTALLER_ID> --devices=<LIST_OF_DEVICE_UUIDS>`
- Requests the installer entry from `macos_installers`.
- Uses `manifest`, `url_token` to generate a [InstallEnterpriseApplication command](https://developer.apple.com/documentation/devicemanagement/installenterpriseapplicationcommand/command) does not support authentication on the package URL specified in the manifest.
- Sends the command to the listed devices.