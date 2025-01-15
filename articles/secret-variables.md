# How to use secret variables in Fleet

Fleet [v4.62.0](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.62.0) allows you to use secret variables in Fleet scripts, software install/uninstall scripts, and MDM configuration profiles. Secret variables are encrypted and stored securely in Fleet, enabling you to use sensitive information in your scripts and profiles without exposing it in plain text. Fleet secret variables cannot be retrieved via the Fleet API or UI.

Examples of sensitive information include:
- API tokens
- Passwords
- Certificates
- Private keys
- Other sensitive data

## Prerequisites

- Fleet v4.62.0

## How to specify a secret variable

A secret variable can be used in a script or MDM configuration profile by specifying the variable name in the format `$FLEET_SECRET_MYNAME` or `${FLEET_SECRET_MYNAME}`. When the script or profile is sent to the host, Fleet will replace the variable with the actual secret value. The prefix `FLEET_SECRET_` is required to indicate that the variable is a secret, and Fleet reserves this prefix for secret variables.

**Example:**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>PayloadDisplayName</key>
    <string>Certificate PKCS12</string>
    <key>PayloadIdentifier</key>
    <string>com.example.certificate</string>
    <key>PayloadType</key>
    <string>Configuration</string>
    <key>PayloadUUID</key>
    <string>918ee83d-ebd5-4192-bcd4-8b4feb750e4b</string>
    <key>PayloadVersion</key>
    <integer>1</integer>
    <key>PayloadContent</key>
    <array>
      <dict>
            <key>Password</key>
            <string>$FLEET_SECRET_CERT_PASSWORD</string>
            <key>PayloadContent</key>
            <data>$FLEET_SECRET_CERT_BASE64</data>
            <key>PayloadDisplayName</key>
            <string>Certificate PKCS12</string>
            <key>PayloadIdentifier</key>
            <string>com.example.certificate</string>
            <key>PayloadType</key>
            <string>com.apple.security.pkcs12</string>
            <key>PayloadUUID</key>
            <string>25cdd076-f1e7-4932-aa30-1d4240534fb0</string>
            <key>PayloadVersion</key>
            <integer>1</integer>
        </dict>
    </array>
</dict>
</plist>
```

In the example above, we use `$FLEET_SECRET_CERT_PASSWORD` and `$FLEET_SECRET_CERT_BASE64` secret variables.

## Using secret variables with GitOps

You can configure Fleet using the [best practice GitOps workflow](https://fleetdm.com/docs/configuration/yaml-files).

You must add the secret variables to your repository's secrets to use them in GitOps.

For the GitHub GitOps flow, they must also be added to the `env` section of your workflow file, as shown below:

```yaml
    env:
      FLEET_URL: ${{ secrets.FLEET_URL }}
      FLEET_API_TOKEN: ${{ secrets.FLEET_API_TOKEN }}
      FLEET_GLOBAL_ENROLL_SECRET: ${{ secrets.FLEET_GLOBAL_ENROLL_SECRET }}
      FLEET_WORKSTATIONS_ENROLL_SECRET: ${{ secrets.FLEET_WORKSTATIONS_ENROLL_SECRET }}
      FLEET_WORKSTATIONS_CANARY_ENROLL_SECRET: ${{ secrets.FLEET_WORKSTATIONS_CANARY_ENROLL_SECRET }}
      FLEET_SECRET_CERT_PASSWORD: ${{ secrets.FLEET_SECRET_CERT_PASSWORD }}
      FLEET_SECRET_CERT_BASE64: ${{ secrets.FLEET_SECRET_CERT_BASE64 }}
```

When GitOps syncs the configuration, it looks for secret variables in scripts and profiles, extracts the secret values from the environment, and uploads them to Fleet.

On subsequent GitOps syncs, if a secret variable used by a configuration profile has been updated, the profile will be resent to the host device(s).

_Note:_ Profiles with secret variables are not entirely validated during a GitOps dry run because secret variables may not be present/correct in the database during the dry run. Hence, there is an increased chance of GitOps non-dry run failure when using a profile with a secret variable. Try uploading this profile to a test team first.

## Using secret variables with the Fleet API and UI

Before uploading a script/profile with secret variables via the Fleet API or UI, you must create the secret variables in Fleet. You can do this with the new secret variables API endpoint.

**Example:**

```bash
curl \
-H "Authorization: Bearer $FLEET_API_TOKEN" \
-H 'Content-Type: application/json' \
https://fleet.example.com/api/v1/fleet/spec/secret_variables \
-X PUT --data-binary @- << EOF
{ "secrets":
 [
  {
    "name": "FLEET_SECRET_CERT_PASSWORD",
    "value": "abc123"
  },
  {
    "name": "FLEET_SECRET_CERT_BASE64",
    "value": "SGVsbG8gV29ybGQh"
  }
 ]
}
EOF
```

Afterward, you can upload the script/profile with secret variables via the Fleet API or UI.

_Note:_ The checksum of Apple DDM profiles with secret variables now includes the timestamp of the last secrets update.

## Escaping variable and interpolation

The dollar sign ($) can be escaped so it's not considered a variable by using a backslash (e.g. `\$100`). Additionally, `MY${variable}HERE` syntax can be used to put strings around the variable.

## Known limitations and issues

- Windows profiles are currently not re-sent to the device on fleetctl gitops update: [issue #25030](https://github.com/fleetdm/fleet/issues/25030)
- Fleet does not mask the secret in script results. DO NOT print/echo your secrets to the console output.
- There is no way to explicitly delete a secret variable. Instead, you can overwrite it with any value.
- Do not use deprecated API endpoint(s) to upload profiles containing secret variables. Use endpoints documented in [Fleet's REST API](https://fleetdm.com/docs/rest-api/rest-api).

<meta name="articleTitle" value="How to use secret variables in Fleet">
<meta name="authorFullName" value="Victor Lyuboslavsky">
<meta name="authorGitHubUsername" value="getvictor">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-01-02">
<meta name="description" value="A guide on using secret variables in Fleet scripts and MDM configuration profiles.">
