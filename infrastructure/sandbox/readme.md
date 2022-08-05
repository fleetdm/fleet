## Terraform for the Fleet Demo Environment
This folder holds the infrastructure code for Fleet's demo environment. See https://github.com/fleetdm/fleet-infra/pull/3 for design documentation.

The interface into this code is designed to be minimal.
If you require changes beyond whats described here, contact @zwinnerman-fleetdm.

### Deploying your code to the loadtesting environment
1. Initialize your terraform environment with `terraform init`
2. Check out the appropiate workspace for your code, for instance `terraform workspace select production`
3. Apply terraform with your branch name with `terraform apply -var tag=BRANCH_NAME -var-file production.tfvars`

### Bugs
1. module.shared-infrastructure.kubernetes_manifest.targetgroupbinding is bugged sometimes, if it gives issues just comment it out
1. on a fresh apply, module.shared-infrastructure.aws_acm_certificate.main will have to be targeted first, then a normal apply can follow
1. If errors happen, see if applying again will fix it

### Maintenance commands
#### Referesh fleet instances
```bash
for i in $(aws dynamodb scan --table-name sandbox-prod-lifecycle | jq -r '.Items[] | select(.State.S == "unclaimed") | .ID.S'); do helm uninstall $i; aws dynamodb delete-item --table-name sandbox-prod-lifecycle --key "{\"ID\": {\"S\": \"${i}\"}}"; done
```

#### Cleanup instances that are running, but not tracked
```bash
for i in $((aws dynamodb scan --table-name sandbox-prod-lifecycle | jq -r '.Items[] | .ID.S'; aws dynamodb scan --table-name sandbox-prod-lifecycle | jq -r '.Items[] | .ID.S'; helm list | tail -n +2 | cut -f 1) | sort | uniq -u); do helm uninstall $i; done
```

#### Cleanup instances that failed to provision
```bash
for i in $(aws dynamodb scan --table-name sandbox-prod-lifecycle | jq -r '.Items[] | select(.State.S == "provisioned") | .ID.S'); do helm uninstall $i; aws dynamodb delete-item --table-name sandbox-prod-lifecycle --key "{\"ID\": {\"S\": \"${i}\"}}"; done
```
