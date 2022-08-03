## Terraform for the Fleet Demo Environment
This folder holds the infrastructure code for Fleet's demo environment.

### Bugs
1. module.shared-infrastructure.kubernetes_manifest.targetgroupbinding is bugged sometimes, if it gives issues just comment it out
1. on a fresh apply, module.shared-infrastructure.aws_acm_certificate.main will have to be targeted first, then a normal apply can follow
1. If errors happen, see if applying again will fix it
1. There is a secret for apple signing whos values are not provided by this code. If you destroy/apply this secret, then it will have to be filled in manually.

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
