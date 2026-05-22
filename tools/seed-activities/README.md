# seed-activities

Inserts one row for every activity type into a local Fleet database so the
activity feeds and host activity cards can be reviewed in the UI without
driving each flow by hand.

Writes directly to `activity_past` (and `activity_host_past` for
host-scoped activities). The service layer is bypassed, so no webhooks
fire and no other side effects occur — safe to run repeatedly against a
dev DB.

## Usage

From the repo root:

```bash
go run ./tools/seed-activities
go run ./tools/seed-activities -host-id 3 -actor admin@example.com
go run ./tools/seed-activities -self-service-only
```

## Examples

Seed one row of every activity type against the default local DB, linking
host-scoped activities to host id 3:

```bash
go run ./tools/seed-activities -host-id 3
```

Review what the My device API writes — six rows (one per install/uninstall
status) with `NULL` `user_id` and empty `user_email`:

```bash
go run ./tools/seed-activities -host-id 5 -self-service-only
```

Then refresh the dashboard activity feed, or open host 3 / host 5's
activity card, to inspect the rendered copy.

## Flags

| Flag | Default | Purpose |
|------|---------|---------|
| `-dsn` | `root:toor@tcp(127.0.0.1:3306)/fleet?...` | MySQL DSN |
| `-actor` | `admin@example.com` | `user_email` recorded on seeded rows |
| `-actor-name` | `Test Admin` | `user_name` recorded on seeded rows |
| `-actor-id` | `1` | `user_id` recorded on seeded rows (must exist in `users`) |
| `-host-id` | `3` | Host id used for host-scoped activities and `activity_host_past` links |
| `-self-service-only` | `false` | Seed only self-service software install/uninstall activities, one row per status variant (mimics the My device API: NULL `user_id`, empty `user_email`) |

## Adding new activity types

`seedActivities` in `activities.go` lists one zero-value instance of every
activity type in `server/fleet/activities.go`. When upstream adds a new
activity type, regenerate the list — see the comment in `activities.go`
for the one-liner.
