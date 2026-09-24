> [!NOTE]
> **Prefer [`dibble`](../../../dibble/README.md) for seeding teams + Apple MDM
> profiles.** The equivalents are:
> ```bash
> ./tools/dibble/dibble teams --count N
> ./tools/dibble/dibble profiles --count N    # Apple .mobileconfig + Windows .xml
> ```
> This tool is kept for backwards compatibility — it focuses on Apple MDM
> enrollment load and uses Apple-specific assets dibble doesn't. We'll
> consolidate when dibble's MDM enrollment path is finished.

# Apple MDM load test

Loadtest harness for Apple MDM enrollment. Creates teams, attaches enrollment
profiles, and simulates enrollment churn.

## Usage

```bash
go run ./tools/mdm/apple/loadtest \
  -fleet_url https://localhost:8080 \
  -api_token "$FLEET_API_TOKEN" \
  -team_count 50 \
  -loop_count 1
```

See `loadtest.go` for the full set of flags.
