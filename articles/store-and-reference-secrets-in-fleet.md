# Store and reference secrets in Fleet

Scripts, configuration profiles, and host name templates sometimes need a sensitive value, like an API token, a license key, or a password, but pasting that value directly into a script or a GitOps YAML file puts it in Git history and in front of anyone with read access to the repo or the Fleet UI. Fleet secrets solve this: store the value once, then reference it by name wherever you need it. Fleet keeps the value encrypted and only ever reveals it to the host at the moment a script runs or a profile installs.

> **Note:** Fleet secrets (`$FLEET_SECRET_*`) are a different thing from [enroll secrets](https://fleetdm.com/docs/using-fleet/enroll-hosts) (the `secrets` key in `org_settings` or a fleet's YAML). Enroll secrets authenticate a host enrolling into Fleet. Fleet secrets are custom values you define and reference in scripts, profiles, and name templates.

## Prerequisites

- A Fleet global admin or maintainer role to create or delete secrets.
- If you're self-hosting Fleet, [`server_private_key`](https://fleetdm.com/docs/configuration/fleet-server-configuration#server-private-key) configured. Fleet uses it to encrypt secret values before storing them. Fleet-managed cloud deployments already have this set.

> **Warning:** Anyone who can read a script, profile, or name template that references a secret can trigger that secret's value to be sent to a host. Scope who has access to author scripts, profiles, and templates accordingly. A secret's value is never shown again in the Fleet UI or API after you create it.

## Create a secret

### In the Fleet UI

1. Go to **Controls > Variables**.
2. Select **Add variable**.
3. Enter a **Name** and **Value**. Fleet uppercases the name and only allows letters, numbers, and underscores. Don't include the `FLEET_SECRET_` prefix. Fleet adds it for you, so a variable named `LICENSE_TOKEN` becomes `$FLEET_SECRET_LICENSE_TOKEN` wherever you reference it.
4. Select **Save**.

### Via the API

```
POST /api/v1/fleet/custom_variables
```

```json
{
  "name": "LICENSE_TOKEN",
  "value": "971ef02b93c74ca9b22b694a9251f1d6"
}
```

See the [custom variables API reference](https://fleetdm.com/docs/rest-api/rest-api#create-custom-variable) for the full request and response.

> **Note:** If you get a "Missing required private key" error, [set `server_private_key`](https://fleetdm.com/docs/configuration/fleet-server-configuration#server-private-key) before creating a secret.

### With GitOps

If you manage Fleet through [GitOps](https://fleetdm.com/docs/configuration/yaml-files), don't create secrets ahead of time in the UI or API. Instead, set an environment variable with the same name as the secret, including the `FLEET_SECRET_` prefix, in the environment where you run `fleetctl gitops`:

```bash
export FLEET_SECRET_LICENSE_TOKEN="971ef02b93c74ca9b22b694a9251f1d6"
fleetctl gitops -f teams/workstations.yml
```

`fleetctl gitops` scans every script, configuration profile, `name_template`, and software install/uninstall/post-install script in your YAML for `$FLEET_SECRET_*` references, looks up each one as a local environment variable, and syncs it to Fleet before applying the rest of the run. The placeholder itself stays in the file. Only the value `fleetctl gitops` reads from the environment gets sent to Fleet, so the copy of the script or profile committed to Git never contains the secret.

Two things behave differently here than creating a secret through the UI:

- The environment variable must be set every time you run `fleetctl gitops`, even if the secret already exists in Fleet from a previous run. If it's missing, the run fails immediately with `environment variable "FLEET_SECRET_LICENSE_TOKEN" not set`, before anything is applied.
- Re-running with a changed value updates the secret. Unlike the UI, which blocks creating a duplicate name, GitOps upserts: change the environment variable's value and run `fleetctl gitops` again to rotate it.

In CI, store the value as a repository secret and expose it to the `fleetctl gitops` step under the `FLEET_SECRET_` name it's referenced by.

If your repo was scaffolded with `fleetctl new`, add it to the `env:` block in `.github/workflows/workflow.yml`, alongside `FLEET_URL` and `FLEET_API_TOKEN`:

```yaml
env:
  FLEET_URL: ${{ secrets.FLEET_URL && secrets.FLEET_URL || 'https://fleet.example.com' }}
  FLEET_API_TOKEN: ${{ secrets.FLEET_API_TOKEN }}
  FLEET_SECRET_LICENSE_TOKEN: ${{ secrets.LICENSE_KEY }}
```

`secrets.LICENSE_KEY` must match the name of the repository secret you stored the value under in GitHub. It doesn't need to match the environment variable name on the left, which is fixed by the `$FLEET_SECRET_LICENSE_TOKEN` reference in your script or profile and must include the `FLEET_SECRET_` prefix exactly.

On GitLab, `.gitlab-ci.yml` doesn't need editing: add the secret under **Settings > CI/CD > Variables** using the `FLEET_SECRET_` name, and GitLab exposes it to the job as an environment variable automatically.

> **Note:** If your YAML references any `$FLEET_SECRET_*` variable, `server_private_key` must be set before you run `fleetctl gitops`, even with `--dry-run`. GitOps validates and syncs secrets before the rest of the run, dry or not, and skips this check entirely if nothing in your YAML references a secret.

## Reference a secret

Once a secret exists, reference it as `$FLEET_SECRET_<NAME>` or `${FLEET_SECRET_<NAME>}` in any of the following. Fleet validates the reference when you save the script, profile, or template, and rejects it if the secret doesn't exist yet.

- **Scripts.** Install, uninstall, and post-install scripts on software packages, and scripts you run directly from **Controls > Scripts**. Fleet substitutes the value when it sends the script to the host.
- **Configuration profiles.** macOS `.mobileconfig`/JSON declarations and Windows `.xml` profiles. Fleet substitutes the value when it delivers the profile.
- **Host name templates** (Fleet Premium). The naming convention applied to macOS, iOS, and iPadOS hosts, set under a fleet's host name template or in `no_team.yml`/`default.yml` controls. Unlike scripts and profiles, a secret used here becomes the host's name, which is visible in the Fleet UI, so don't reference secrets you need to keep hidden.

## Worked example: fetch a file from an authenticated URL at install time

Say you host a per-customer MSI license transform behind an authenticated URL and want a Windows install script to pull it down at install time, without hardcoding the credential in the script.

1. Create a secret named `LICENSE_TOKEN` with the bearer token as its value.
2. Reference it in the install script:

```powershell
$licenseToken = "$FLEET_SECRET_LICENSE_TOKEN"
$transformPath = "$env:TEMP\transform.mst"

Invoke-WebRequest -Uri "https://licenses.example.com/customer-123/transform.mst" `
  -Headers @{ Authorization = "Bearer $licenseToken" } `
  -OutFile $transformPath

Start-Process msiexec.exe -ArgumentList "/i installer.msi TRANSFORMS=`"$transformPath`" /quiet" -Wait
```

Fleet substitutes `$FLEET_SECRET_LICENSE_TOKEN` for the real token before the script ever reaches the host, so by the time PowerShell runs it, the line reads as a plain string assignment. The script itself, and its copy in Git if you manage scripts through GitOps, never contains the token.

## Verify

1. Open the script, profile, or template you referenced the secret in.
2. Confirm it saved without a "missing secret" error. Fleet checks that every `$FLEET_SECRET_*` reference resolves to an existing secret before accepting the save.
3. Run the script or push the profile to a test host and confirm the substituted value behaves as expected, for example by checking the script's own log output for the resulting command, not the secret value itself.

## Troubleshoot

**"Couldn't add. Secret variable ... missing from database."** The script or profile references a secret that doesn't exist yet. Create it under **Controls > Variables** (or via the API) using the exact name after `FLEET_SECRET_`, then save again.

**"Missing required private key."** Set [`server_private_key`](https://fleetdm.com/docs/configuration/fleet-server-configuration#server-private-key) on self-hosted Fleet before creating a secret.

**`fleetctl gitops` fails with `environment variable "FLEET_SECRET_..." not set`.** A script, profile, or `name_template` in your YAML references that secret, but the matching environment variable isn't set in the shell (or CI job) running `fleetctl gitops`. Export it, or add it to your CI step's `env:` block, using the exact name after `$`, including the `FLEET_SECRET_` prefix.

## Further reading

- [Custom variables API reference](https://fleetdm.com/docs/rest-api/rest-api#custom-variables)
- [Configuration profiles YAML reference](https://fleetdm.com/docs/configuration/yaml-files)
- [Fleet server configuration reference](https://fleetdm.com/docs/configuration/fleet-server-configuration)

<meta name="articleTitle" value="Store and reference secrets in Fleet">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="publishedOn" value="2026-08-17">
<meta name="category" value="guides">
<meta name="description" value="Store sensitive values as Fleet secrets, then reference them in scripts, configuration profiles, and host name templates.">
