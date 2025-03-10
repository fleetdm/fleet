# Fleet-maintained apps

## Adding a new FMA

1. Decide on a source for the app's metadata. We currently support homebrew as a source for macOS
   apps.
2. Find that app's metadata. For homebrew, you can visit https://formulae.brew.sh/ and find the app
   there.
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
5. Open a PR to the `fleet` repository with the new 

### JSON schema for input files
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Fleet-maintained app input file schema",
  "type": "object",
  "properties": {
    "name": {
      "description": "The user-facing name of the app.",
      "type": "string"
    },
    "unique_identifier": {
      "description": "The platform-specific unique identifier for this app. On macOS, this is the bundle identifier.",
      "type": "string"
    },
    "source_identifier": {
      "description": "The identifier used by the metadata source for this app. For homebrew, this is the the `token` field.",
      "type": "string"
    },
    "installer_format": {
      "description": "The type of installer used to install the app.",
      "type": "string",
      "enum": ["zip:app", "dmg:app", "dmg:pkg", "pkg"]
    },
    "source": {
      "description": "The metadata source for this app.",
      "type": "string",
      "enum": ["homebrew"]
    }
  },
  "required": [
    "name",
    "unique_identifier",
    "source_identifier",
    "installer_format",
    "source"
  ]
}
```