# Fleet-maintained apps (FMA)

## Adding a new app (macOS)

1. Create a new issue using the [New Fleet-maintained app](https://github.com/fleetdm/fleet/issues/new?template=fma-request.md) issue template
2. Find the app's metadata in its [Homebrew formulae](https://formulae.brew.sh/)
3. Create a new mainfiest file called `$YOUR_APP_NAME.json` in the `inputs/homebrew/` directory. For
   example, if you wanted to add Box Drive, create the file `inputs/homebrew/box-drive.json`.
4. Fill out the file according to the [input schema below](#input-file-schema). For our example Box Drive app, it would look like this:

   ```json
   {
      "name": "Box Drive",
      "slug": "box-drive/darwin",
      "unique_identifier": "com.box.desktop",
      "token": "box-drive",
      "installer_format": "pkg",
      "default_categories": ["Productivity"]
   }
   ```

5. Run the following command from the root of the Fleet repo to generate the app's output data:

    ```bash
   go run cmd/maintained-apps/main.go --slug="<slug-name>" --debug
   ```

6. Open a PR to the `fleet` repository with the above changes.  Connect it to the issue by adding `Fixes #ISSUE_NUMBER` in the description.

7. The [#g-software product group](https://fleetdm.com/handbook/company/product-groups#software-group) will:
   1. Review the PR and test the app.  Contributors should be aware of the validation requirements below.
   2. If validation requirements cannot be met in this PR, the PR will be closed and the associated issue will be prioritized in the g-software group backlog.

8. If the app passes testing, it is approved and merged. The app should appear shortly in the Fleet-maintained apps section when adding new software to Fleet. The app icon will not appear in Fleet until the following release. App icon progress is tracked in the issue. An addition to Fleet-maintained apps is not considered "Done" until the icon is added in a Fleet release. This behavior will be [improved](https://github.com/fleetdm/fleet/issues/29177) in a future release.

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

Use `darwin` as the platform part.  For example, use a `slug` of `box-drive/darwin` for Box Drive on macOS.

#### `pre_uninstall_scripts` (optional)

These are command lines that will be run _before_ the generated uninstall script is executed, e.g. for [Box](inputs/homebrew/box-drive.json).

#### `post_uninstall_scripts` (optional)

These are command lines that will be run _after_ the generated uninstall script is executed, e.g. for [Box](inputs/homebrew/box-drive.json).

#### `default_categories` (required)

These are the default categories assigned to the installer in [self-service](https://fleetdm.com/guides/software-self-service) if no categories are specified when it is added to a team's library.  Categories must be one or more of these values:

- `Browsers`
- `Communication`
- `Developer Tools`
- `Productivity`

### Validating new additions to the Fleet-maintained apps

1. When a pull request (PR) is opened containing changes to `ee/maintained-apps/inputs/`, the [#g-software Product Designer (PD) and Engineering Manager (EM)](https://fleetdm.com/handbook/company/product-groups#software-group) are automatically added as reviewers.
   1. The PD is responsible for approving the name and default category
   2. The EM is repsonsible for validating or assigning a validator

2. Ensure an associated issue exists for the PR.  If not create one using the `Add Fleet-maintained app` issue template, move the issue to the `g-software` project and set the status to `In Review`.  Ensure the PR is linked to the issue.

3. Validate the PR:

   1. Find the app in [Homebrew's GitHub casks](https://github.com/Homebrew/homebrew-cask/tree/main/Casks) and download it locally using `cask.url`.
   2. Install it on a host and run a live query on the host: `SELECT * FROM apps WHERE name LIKE '%App Name%';`
   3. Validate and check off items in the `Validation` section of the issue.


4. If the PR passes validation, set the issue status to `Awaiting QA`.  The validator will also perform the test criteria and check off QA tasks in the issue.  If tests fail, add feedback in the PR comments.  If the test failure(s) cannot be addressed by the contributor, close the PR and move the issue to the Drafting board for prioritization.  If tests pass, the PR is approved and merged.

5. Product designer is notified in the issue to add this icon to Fleet's [design system in Figma](https://www.figma.com/design/8oXlYXpgCV1Sn4ek7OworP/%F0%9F%A7%A9-Design-system?node-id=264-2671) and publish the icon as a part of the Software icon Figma component.

6. The validator is responsible for adding the icon to Fleet (e.g. the TypeScript components of [#29175](https://github.com/fleetdm/fleet/pull/29175/files))

7. QA ensures the icon is added to Fleet

8. @eashaw is notified in the issue to add the icon to the website [fleetdm.com/app-library](https://fleetdm.com/app-library).

#### Testing additions to Fleet-maintained apps (no icon)

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

If testing fails:

- Remove issue from the `g-software` release board, and add issue to the `Drafting` board. Remove the `:release` tag and add the `:product` tag.

## Updating existing Fleet-maintained apps

Fleet-maintained apps need to be updated as frequently as possible while maintaining reliability.  This is currently a balancing act as both scenarios below result in customer workflow blocking bugs:

- App vendor updates to installers can break install/uninstall scripts
- App vendors will deprecate download links for older installers

A Github action periodically creates a PR that updates one or more apps in the catalog by:

- Bumping versions
- Regenerating install/uninstall scripts

Each app updated in the PR must be validated independently.  Only merge the PR if all apps changed meet the following criteria:

- [X] App can be downloaded using manifest URL
- [X] App installs successfully on host using manifest install script
- [X] App exists on host
- [X] App uninstalls successfully on host using manifest uninstall script

If an app does not pass test criteria:

- [Freeze the app](#freezing-an-existing-app)
- File a bug for tracking

## Freezing an existing Fleet-maintained app

If any app fails validation:

1. Do not merge the PR as-is.
2. Add `"frozen": true"` to the failing app's input file (e.g.,`inputs/homebrew/<app>.json`).
3. Revert its corresponding output manifest file (e.g., `outputs/<slug>.json`) to the version in the `main` branch:

   ```bash
   git checkout origin/main -- ee/maintained-apps/outputs/<slug>.json
   ```

4. Validate changes in the frozen input file by running the following.  This should output no errors and generate no changes.

   ```bash
   go run cmd/maintained-apps/main.go --slug="<slug>" --debug
   ```

5. Commit both the input change and the output file revert to the same PR.


