# Fleet-maintained apps

## Adding a new FMA

1. Decide on a source for the app's metadata. We currently support homebrew as a source for macOS apps.
2. Find that app's metadata. For homebrew, you can visit https://formulae.brew.sh/ and find the app there.
3. Create a new file called `$YOUR_APP_NAME.json` in the `inputs/$SOURCE` directory. For
   example, if you wanted to add Box Drive and use homebrew as the source, you would create the
   file `inputs/homebrew/box-drive.json`.
4. Fill out the file according to the [breakdown below](#input-file-schema). For our example Box Drive app, it would look like this:
   ```json
   {
      "name": "Box Drive",
      "slug": "box-drive/darwin",
      "unique_identifier": "com.box.desktop",
      "token": "box-drive",
      "installer_format": "pkg",
      "pre_uninstall_scripts": [
         "(cd /Users/$LOGGED_IN_USER; sudo -u $LOGGED_IN_USER fileproviderctl domain remove -A com.box.desktop.boxfileprovider)",
         "(cd /Users/$LOGGED_IN_USER; sudo -u $LOGGED_IN_USER /Applications/Box.app/Contents/MacOS/fpe/streem --remove-fpe-domain-and-archive-unsynced-content Box)",
         "(cd /Users/$LOGGED_IN_USER; sudo -u $LOGGED_IN_USER /Applications/Box.app/Contents/MacOS/fpe/streem --remove-fpe-domain-and-preserve-unsynced-content Box)",
         "(cd /Users/$LOGGED_IN_USER; defaults delete com.box.desktop)",
         "echo \"${LOGGED_IN_USER} ALL = (root) NOPASSWD: /Library/Application\\ Support/Box/uninstall_box_drive_r\" >> /etc/sudoers.d/box_uninstall"
      ],
      "post_uninstall_scripts": ["rm /etc/sudoers.d/box_uninstall"]
   }
   ```
5. Open a PR to the `fleet` repository with the new app file. This will trigger a CI job which will automatically update your PR with the required output files. These files contain important data such as the install and uninstall scripts for the app.
6. A Fleetie will test and review the PR. Once approved and merged, the app should appear in the Fleet-maintained apps section when adding new software to Fleet.

### Input file schema

#### `name`
This is the user-facing name of the application.

#### `unique_identifier`
This is the platform-specific unique identifier for the app. On macOS, this is the app's bundle identifier.

#### `token`
This is the identifier used by homebrew for the app; it is the `token` field on the homebrew API response.

#### `installer_format`
This is the file format for the app's installer. Currently supported values are:
- `zip`
- `dmg`
- `pkg`

To find the app's installer format, you can look at the `url` field on the homebrew API response. The installer's extension should be at the end of this URL. 

Sometimes the file type is not included in the installer's URL. In this case, you can download the installer and use the extension of the downloaded file.

#### `slug`
The `slug` identifies a specific app and platform combination. It is used to name the manifest files that contain the metadata that Fleet needs to add, install, and uninstall this app. 

The slug is composed of a filesystem-friendly version of the app name, and an operating system platform identifier, separated by a `/`.

For the app name part, use `-` to separate words if necessary, for example `adobe-acrobat-reader`. 

The platform part can be any of these values:
- `darwin`

For example, use a `slug` of `box-drive/darwin` for Box Drive on macOS.

#### `pre_uninstall_scripts`
These are command lines that will be run _before_ the generated uninstall script is executed.

#### `post_uninstall_scripts`
These are command lines that will be run _after_ the generated uninstall script is executed.
