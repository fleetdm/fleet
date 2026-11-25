# Activity driver for a load test

Runs the load-test activity driver (`tools/loadtest/qa/driver`) as one small ECS task next to a load test: the admin
and API workload all day, and the checks that watch the instance. It's the same binary that runs from a laptop, with
the laptop's files replaced:

| On a laptop | In ECS |
|---|---|
| fleetctl context's token | Secrets Manager (`token`, set by hand) |
| private profile file | Secrets Manager (`profile`, set by hand) |
| `~/.fleet-driver/<host>/users.json` | passwords derived from a generated secret (`-users_secret_env`) |
| `STOP` file | `s3://<bucket>/STOP` |
| events file and `driver.log` | the `fleet-<workspace>-activity-driver` log group (events are the JSON lines) |
| `-on_trip` script | a webhook (`notify = true`, Slack incoming-webhook URL set by hand) |
| local GitOps repo | a tarball in the bucket, with its own `apply.sh` |

The task reads the instance (ECS service, CloudWatch metrics, Logs Insights on the Fleet log group) and its own
bucket; it can't change Fleet's infrastructure. It calls Fleet's admin API through the internal load balancer.

## Deploy

Use the load test's terraform workspace name (the infra root's), so the driver finds its cluster, log group and
Aurora cluster.

```sh
terraform init
terraform workspace select <workspace>      # or: terraform workspace new <workspace>
terraform apply -var git_ref=<load test tag or branch> -var desired_count=0
```

Then set the secrets' values (`terraform output secrets` names them), without putting them in a shell history:

```sh
# An admin API-only user's token (create the user with fleetctl user create --api-only --global-role admin).
aws secretsmanager put-secret-value --secret-id <token secret> --secret-string file://token.txt
# The private profile YAML. Never commit it.
aws secretsmanager put-secret-value --secret-id <profile secret> --secret-string file://profile.yaml
```

and start it:

```sh
terraform apply -var git_ref=<same> -var desired_count=1
```

### GitOps runs

`gitops.apply` (W-GITOPS-01) needs the instance's GitOps repo, which is private (it's generated from the customer
profile). Pack it with an `apply.sh` at its top that runs the apply (for example
`fleetctl gitops -f default.yml -f unassigned.yml -f fleets/a.yml ...`, big fleets last), upload it, and point the
task at it:

```sh
tar -czf gitops.tar.gz -C <repo> .
aws s3 cp gitops.tar.gz s3://$(terraform output -raw bucket)/gitops/gitops.tar.gz
terraform apply -var git_ref=<same> -var gitops_bundle_key=gitops.tar.gz
```

The task unpacks it at start and configures fleetctl with the driver's token. Edits the driver makes to the repo
live until the task restarts. Without a bundle, W-GITOPS-01 is dropped at start and everything else runs.

## Operate

- **Pause / resume:** the workload is paused while `s3://<bucket>/STOP` exists (`terraform output stop_marker`). A
  trip creates it with the reason as its first line; checks keep reading while paused. Recover by hand, then
  `aws s3 rm` it.
- **Stop:** `terraform apply ... -var desired_count=0`.
- **Events:** Logs Insights on the driver's log group, e.g.
  `filter kind = "action" | stats count() by id, verdict` or `filter kind = "check" and verdict = "breach"`.
- **Cost:** a 0.5 vCPU / 2 GB Fargate task (about $0.70 a day) plus a few cents of Logs Insights scans.

Destroy this root before the load test's infra root.
