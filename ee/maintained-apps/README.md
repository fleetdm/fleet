# Fleet-maintained apps (FMA)

## Freezing an existing app

Add `"frozen": true` to the appropriate input JSON file to pause automated updates to the corresponding output manifest.
To aid in testing, manifests will still be generated for frozen inputs if the output file doesn't exist.

Apps should be frozen when updating the manifest would introduce regressions on ability to install/uninstall the app.
Frozen apps should have bugs filed, and fixes for those bugs should unfreeze the app and bump it to the latest version
as part of the fix PR.

## Adding a new app

1. Decide on a source for the app's metadata. We currently support homebrew as a source for macOS apps.
2. Find that app's metadata. For homebrew, you can visit [Homebrew formulae](https://formulae.brew.sh/) and find the app there.
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

5. Run the following command to generate output data:

    ```bash
   go run cmd/maintained-apps/main.go -slug="box-drive/darwin" -debug
   ```

6. Open a PR to the `fleet` repository with the above changes

7. The [#g-software product group](https://fleetdm.com/handbook/company/product-groups#software-group) will:
   1. Review the PR and test the app.  Contributors should be aware of the validation requirements below.
   2. Create an associated issue to track progress and icon additions.
   3. If validation requirements cannot be met in this PR, the PR will be closed and the associated issue will be prioritized in the g-software group backlog.

8. If approved and merged, the app should appear shortly in the Fleet-maintained apps section when adding new software to Fleet.  The app icon will not appear in Fleet until a following release.  An FMA addition is not considered "Done" until the icon is added in a Fleet release. This behavior will be [improved](https://github.com/fleetdm/fleet/issues/29177) in a future release.

### Input file schema

#### `name` (required)

This is the user-facing name of the application.

#### `unique_identifier` (required)

This is the platform-specific unique identifier for the app. On macOS, this is the app's bundle identifier.

#### `token` (required)

This is the identifier used by homebrew for the app; it is the `token` field on the homebrew API response.

#### `installer_format` (required)

This is the file format for the app's installer. Currently supported values are:

- `zip`
- `dmg`
- `pkg`

To find the app's installer format, you can look at the `url` field on the homebrew API response. The installer's extension should be at the end of this URL.

Sometimes the file type is not included in the installer's URL. In this case, you can download the installer and use the extension of the downloaded file.

#### `slug` (required)

The `slug` identifies a specific app and platform combination. It is used to name the manifest files that contain the metadata that Fleet needs to add, install, and uninstall this app.  This is what is used when referring to the app in GitOps.

The slug is composed of a filesystem-friendly version of the app name, and an operating system platform identifier, separated by a `/`.

For the app name part, use `-` to separate words if necessary, for example `adobe-acrobat-reader`. 

The platform part can be any of these values:

- `darwin`

For example, use a `slug` of `box-drive/darwin` for Box Drive on macOS.

#### `pre_uninstall_scripts` (optional)

These are command lines that will be run _before_ the generated uninstall script is executed.

#### `post_uninstall_scripts` (optional)

These are command lines that will be run _after_ the generated uninstall script is executed.

#### `default_categories` (required)

These are the default categories assigned to the installer in self-service if no categories are specified when it is added to a team's library.  Categories can contain any of these values:

- `Browsers`
- `Communication`
- `Developer Tools`
- `Productivity`

These are the default categories assigned to the installer in self-service if no categories are specified when it is added to a team's library.  Categories can contain any of these values:

### Validating new additions to the FMA catalog

1. When a pull request (PR) is opened containing changes to `ee/maintained-apps/inputs/`, the [#g-software Product Designer (PD)](https://fleetdm.com/handbook/company/product-groups#software-group) is automatically added as reviewer and is responsible for approving the PR if the FMA name is user-friendly.

2. Create a github issue. This issue will be used to track status and icon additions:
   - add the `:release` label
   - add it to the `g-software` project (status: Ready)
   - link the PR
   - add the below FMA issue template

3. Validate the PR:

   1. Find the app in [Homebrew's GitHub casks](https://github.com/Homebrew/homebrew-cask/tree/main/Casks) and download it locally using `cask.url`.

   2. Install it on a host and run a live query on the host: `SELECT * FROM apps WHERE name LIKE '%App Name%';`
   3. `name` in the inputs manifest matches osquery `app.name`
   4. `unique_identifier` in the inputs manifest matches osquery `app.bundle_identifier`
   5. The version scheme in the homebrew recipe matches osquery `app.bundle_short_version`.  This ensures the FMA can be patched correctly.
   6. Ensure associated outputs were generated in the PR
      - `/outputs/<app-name>/darwin.json` created
      - `/outputs/apps.json` updated

4. If the PR passes validation, set the issue status to `Awaiting QA`.  QA will perform the test criteria.  PR is approved and merged.

5. If tests pass, @eashaw and a Product Designer are added to the PR. Eric adds the icon for [fleetdm.com/app-library](https://fleetdm.com/app-library).

6. Product designer to add this icon to Fleet's [design system in Figma](https://www.figma.com/design/8oXlYXpgCV1Sn4ek7OworP/%F0%9F%A7%A9-Design-system?node-id=264-2671) and publish the icon as a part of the Software icon Figma component.

#### Testing FMA catalog additions (no icon)

Use the `FLEET_DEV_MAINTAINED_APPS_BASE_URL` environment variable with the following value:

   ```bash
   https://raw.githubusercontent.com/<repository-name>/fleet/refs/heads/<PR-branch-name>/ee/maintained-apps/outputs
   ```

   Make sure you replace the `<PR-branch-name>` and `<repository-name>`

Test criteria:

- [X] App adds successfully to team's library
- [X] App installs successfully on host
- [X] App opens succuessfully on host
- [X] App uninstalls successfully on host

If the tests pass:

- Move issue to `Ready` (icon addition still needed)
- Approve and merge PR

If the test fail:
- Remove issue from the `g-software` board, and add issue to the `Drafting` board
