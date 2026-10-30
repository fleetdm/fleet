# Secrets and environment variables

The export never writes a secret value to disk. Each sensitive value becomes a `$VARIABLE` reference that `fleetctl gitops` fills from the environment (CI secrets or a local `.env`). `place` lists every variable in `.env.example`, in the README's "Environment variables" section, and in the GitHub workflow's `env:` block.

For the dry run, `validate` builds the values in memory and passes them only to the `fleetctl` process:

| Source | Meaning |
|---|---|
| **API** | Read from Fleet with a GET (the value is identical to what Fleet has, so the dry run compares like with like) |
| **Mask** | Fleet only returns `********` for this value. The dry run uses that mask, which Fleet treats as "keep the stored value". In CI the variable needs the real value. |
| **User** | Fleet never returns it. Taken from the user's shell if set; otherwise a placeholder, and the report says those files weren't checked with the real value. |

A value the user already exported in their shell wins over the mask for masked values.

| Value | Variable | Where | Dry-run source |
|---|---|---|---|
| Fleet server URL (`server_settings.server_url`) | `FLEET_URL` (the template's) | `default.yml` | API |
| Global enroll secrets | `FLEET_GLOBAL_ENROLL_SECRET`, `_2`, … | `default.yml` `org_settings.secrets` | API (`/spec/enroll_secret`) |
| Fleet enroll secrets | `FLEET_<FLEET>_ENROLL_SECRET`, `_2`, … | fleet file `settings.secrets` | API (`/fleets/{id}`) |
| Webhook destination URLs (activities, host status, failing policies, vulnerabilities, host activities) | `FLEET_<KIND>_URL`, `FLEET_<FLEET>_<KIND>_URL` | `webhook_settings` | API |
| Google Calendar webhook URL (per fleet) | `FLEET_<FLEET>_CALENDAR_WEBHOOK_URL` | fleet `settings.integrations` | API |
| macOS migration webhook URL | `FLEET_MACOS_MIGRATION_WEBHOOK_URL` | `controls.macos_migration` | API |
| Admin SSO IdP metadata XML | `FLEET_SSO_METADATA` | `org_settings.sso_settings.metadata` | API |
| End-user authentication IdP metadata XML | `FLEET_MDM_SSO_METADATA` | `org_settings.mdm.end_user_authentication.metadata` | API |
| Jira and Zendesk API tokens | `FLEET_JIRA_API_TOKEN`, `FLEET_ZENDESK_API_TOKEN`, `_2`, … | `org_settings.integrations` | Mask |
| Google Calendar and Google Workspace service account keys | `FLEET_GOOGLE_CALENDAR_API_KEY_JSON`, `FLEET_GOOGLE_WORKSPACE_API_KEY_JSON` | `org_settings.integrations` | Mask |
| Certificate authority secrets (DigiCert token, NDES password, SCEP challenge, Hydrant client secret, EST password, Smallstep username and password, Google CAS key) | `FLEET_CA_<NAME>_<FIELD>` | `org_settings.certificate_authorities` | API (`include_secrets=true`) |
| EJBCA client PKCS#12 and password | `FLEET_CA_<NAME>_CLIENT_P12`, `FLEET_CA_<NAME>_PASSWORD` | `org_settings.certificate_authorities.ejbca` | Mask (Fleet can't export them; reported as a gap) |
| Apple account provisioning OAuth client secret | `FLEET_APPLE_ACCOUNT_PROVISIONING_CLIENT_SECRET` | `controls.apple_account_provisioning` | Mask |
| Microsoft Graph client secret | — (left out; Fleet keeps the stored secret when the key is absent) | — | — |
| Custom variables in scripts and profiles | `FLEET_SECRET_<NAME>` (name kept) | the script or profile | User |

`SSO_METADATA` values are multi-line XML. `fleetctl gitops` expands variables before parsing YAML, so `place` uncomments the matching lines in `.github/fleet-gitops/gitops.sh` that indent the value (and fixes the template's copy, which reads `FLEET_MDM_SSO_METADATA` for both). `validate` applies the same indentation in memory.

## Not treated as secrets

- SSO and end-user authentication `metadata_url`, entity IDs, IdP names: the template writes them in plain text.
- Values inside profiles and scripts. They're exported exactly as Fleet stores them; `$FLEET_SECRET_*` placeholders stay placeholders. If a profile or script has a credential written into it, flag it to the user rather than changing the file (changing it would change what Fleet deploys).
- GitOps exceptions: when Fleet marks enroll secrets as managed in the UI (**Settings > Integrations > Change management**), no `secrets:` keys are written, because fleetctl fails when they're present. A fresh Fleet instance has this exception turned on.
