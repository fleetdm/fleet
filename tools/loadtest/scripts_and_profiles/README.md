> [!NOTE]
> **Prefer [`dibble`](../../dibble/README.md) for seeding scripts and MDM profiles.** The equivalents are:
> ```bash
> ./tools/dibble/dibble scripts --count N
> ./tools/dibble/dibble profiles --count N
> ```
> This tool is kept for backwards compatibility. We'll remove it once nothing references it.

# scripts_and_profiles

Go program used to load-test scripts and MDM profiles for the
[unified queue story](https://github.com/fleetdm/fleet/issues/22866).

It creates `team_count` teams and attaches a fixed set of scripts and MDM
profiles (Apple + Windows) to each, exercising the scripts/profile distribution
pipeline at scale.

## Usage

```bash
go run ./tools/loadtest/scripts_and_profiles \
  -fleet_url https://localhost:8080 \
  -api_token "$FLEET_API_TOKEN" \
  -team_count 20
```

Add `-cleanup_teams` to delete the created teams after the run.
