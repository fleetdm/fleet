# Fleet-maintained apps (FMA)

## Freezing an existing app

Add `"frozen": true` to the appropriate input JSON file to pause automated updates to the corresponding output manifest.
To aid in testing, manifests will still be generated for frozen inputs if the output file doesn't exist.

Apps should be frozen when updating the manifest would introduce regressions on ability to install/uninstall the app.
Frozen apps should have bugs filed, and fixes for those bugs should unfreeze the app and bump it to the latest version
as part of the fix PR.

## Adding a new app

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
6. The [#g-software product group](https://fleetdm.com/handbook/company/product-groups#software-group) will review the PR and test the app. Once approved and merged, the app should appear in the Fleet-maintained apps section when adding new software to Fleet.

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

### Testing

Fleet tests every Fleet-maintained app. For new apps, start at step 1. For updates to existing apps, skip to step 6.

1. When a pull request (PR) is opened in `inputs/`, the [#g-software Product Designer (PD)](https://fleetdm.com/handbook/company/product-groups#software-group) is automatically added as reviewer.
2. The PD is responsible for making sure that the `name` for the new app matches the name that shows up in Fleet's software inventory. If the name doesn't match or if the name is not user-friendly, the PD will bring it to #g-software design review. This way, when the app is added to Fleet, the app will be matched with the app that comes back in software inventory.
- Find the app in [Homebrew's GitHub casks](https://github.com/Homebrew/homebrew-cask/tree/699414cb220dde2b93af764cea7f24f4018e49ac/Casks) and download it locally using `cask.url`.
- Install it on a host and run a live query on the host: `SELECT * FROM apps WHERE name LIKE '%App Name%';` Ensure the name you get back from the query matches the `name` provided in the PR.

3. Then, the PD builds the app's `outputs/` and updates `outputs/apps.json` on the same PR by running the following command:

```
go run cmd/maintained-apps/main.go -slug="<slug from inputs/type/app-name.json>" -debug
```

4. At this time, @eashaw and a Product Designer are added to the PR. Eric adds the icon for [fleetdm.com/app-library](https://fleetdm.com/app-library).
5. Add an icon for the app to the PR. To add the icon, add an SVG to `frontend/pages/SoftwarePage/components/icons/` and update the `frontend/pages/SoftwarePage/components/icons/index.ts` file. Also add this icon to Fleet's [design system in Figma](https://www.figma.com/design/8oXlYXpgCV1Sn4ek7OworP/%F0%9F%A7%A9-Design-system?node-id=264-2671) and publish the icon as a part of the Software icon Figma component.
6. Then, the PD adds [Quality Assurance (QA)](https://fleetdm.com/handbook/company/product-groups#software-group) as a reviewer. QA is responsible for testing the app. 
7. When testing, update the `FLEET_DEV_MAINTAINED_APPS_BASE_URL` environment variable with the following value:

```
https://raw.githubusercontent.com/fleetdm/fleet/refs/heads/<PR-branch-name>/ee/maintained-apps/outputs
```

Make sure you replace the `<PR-branch-name>`.

8. Add and test the app: Does the icon look right? Does the app install? Does the app uninstall? Can you open the app once it's installed?

9. If the tests fail, the QA sets the PR to draft, files a bug, [triages the bug](https://fleetdm.com/handbook/product-design#triage-new-bugs), and updates the [testing spreadsheet](https://docs.google.com/spreadsheets/d/1H-At5fczHwV2Shm_vZMh0zuWowV7AD7yzHgA0RVN7nQ/edit?gid=0#gid=0).
10. If the test is successful, QA approves and merges the PR.

