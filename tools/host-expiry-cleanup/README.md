# Expired host cleanup runner

The runner divides `host_seen_times.seen_time` into non-overlapping time buckets and feeds them to concurrent workers. Defaults are 30-day buckets and four workers; `--bucket-days` and `--workers` can adjust them. Empty buckets are skipped immediately so a worker can claim the next unassigned bucket. Each worker uses Fleet's datastore expiration logic to delete up to 65,000 hosts per batch; successful 5,000-host delete transactions log the worker ID and cumulative batch total. The runner also removes prior host/activity links for deleted hosts, but intentionally omits the per-host `deleted_host` audit activities and their webhooks for this backfill. Workers share Fleet's `leader` lock for each concurrent round, renew it during long batches, then release it and yield for one minute before the next round. It waits up to 10 minutes for the scheduled cleanup to release the lock before exiting with a timeout. After OSQuery-seen buckets finish, a serial cleanup pass handles MDM-only and never-seen hosts. `--serial-only` skips the seen-time buckets and runs that unscoped pass directly. `--max-rounds 1` bounds a trial to at most one 65,000-host batch per worker; rounds count the serial fallback as well. The build wrapper targets Linux amd64 and runs in the foreground over SSH, so keep the connection open until it exits.

From the Fleet checkout, build and copy the Linux binary to a production host over SSH. The default is a dry run that prints the bucket plan:

```sh
tools/host-expiry-cleanup/run.sh i-0701c76b685c68ebe /tmp/dnrfleet/dnr-fleet-prod-config.us-east-1 --sudo
```

Use `--estimate` to get an optimizer-statistics estimate of rows older than the shortest enabled expiry window. It does not run a full count and does not delete hosts:

```sh
tools/host-expiry-cleanup/run.sh i-0701c76b685c68ebe /tmp/dnrfleet/dnr-fleet-prod-config.us-east-1 --sudo --estimate
```

If the config file is root-readable only, add `--sudo`; it runs the helper with non-interactive sudo. Review the bucket plan, then explicitly add `--apply` to delete hosts. This command is capped at one round:

```sh
tools/host-expiry-cleanup/run.sh i-0701c76b685c68ebe /tmp/dnrfleet/dnr-fleet-prod-config.us-east-1 --sudo --apply --max-rounds 1
```

The config path must be absolute and contain only letters, digits, `/`, `.`, `_`, or `-`. The remote host needs network access to the configured primary database. The runner uses the existing `leader` lock so Fleet's scheduled cleanup can take a turn between batches. Interrupt with Ctrl-C or SIGTERM; active batches finish before it stops.

For a 10-worker trial with weekly buckets, first review the plan without `--apply`:

```sh
tools/host-expiry-cleanup/run.sh i-0701c76b685c68ebe /tmp/dnrfleet/dnr-fleet-prod-config.us-east-1 --sudo --workers 10 --bucket-days 7
```

Then add `--apply --max-rounds 1` to run one bounded round, capped at 650,000 hosts.

To run one bounded batch of the serial MDM-only/never-seen fallback without rescanning the seen-time buckets:

```sh
tools/host-expiry-cleanup/run.sh i-0701c76b685c68ebe /tmp/dnrfleet/dnr-fleet-prod-config.us-east-1 --sudo --serial-only --apply --max-rounds 1
```
