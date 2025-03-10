# Fleet-maintained apps

## Adding a new FMA

1. Decide on a source for the app's metadata. We currently support homebrew as a source for macOS apps.
2. Find that app's metadata. For homebrew, you can visit https://formulae.brew.sh/ and find the app there.
3. Create a new file called `your-app-name.json` in the `inputs/target-platform` directory. For
   example, if you wanted to add Slack for the macOS (aka `darwin`) platform, you would create the
   file `inputs/darwin/slack.json`.
4. Fill out the file according to the [JSON schema below](#json-schema-for-input-files). For our
   example Slack app, it would look like this:
   ```json
   {
        "name": "Slack",
        "unique_identifier": "com.tinyspeck.slackmacgap",
        "source_identifier": "slack",
        "installer_format": "dmg:app",
        "source": "homebrew"
   }
   ```
5. Open a PR to the `fleet` repository with the new app file. This will trigger a CI job which will automatically update your PR with the required output files. These files contain important data such as the install and uninstall scripts for the app.
6. A fleetie will test and review the PR. Once approved and merged, the app should appear in the Fleet-maintained apps section when adding new software to Fleet.

### Input file breakdown

#### `name`
This is the user-facing name of the application.

#### `unique_identifier`
This is the platform-specific unique identifier for the app. On macOS, this is the app's bundle identifier.

#### `source_identifier`
This is the identifier used by the metadata source for this app. For homebrew on macOS, this is the `token` field in the homebrew API response.

#### `installer_format`
This is the file format for the app's installer. Currently supported values are:
- `zip`
- `dmg`
- `pkg`

#### `source`
This is the metadata source for the app. Currently supported values are:
- `"homebrew"`
