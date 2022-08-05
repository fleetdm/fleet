## Terraform for the Fleet Demo Environment
This folder holds the infrastructure code for Fleet's demo environment.

This readme itself is intended for infrastructure developers. If you aren't an infrastructure developer, please see https://sandbox.fleetdm.com/openapi.json for documentation.

### Instance state machine
```
provisioned -> unclaimed -> claimed -> [destroyed]
```
provisioned means an instance was "terraform apply'ed" but no installers were generated.
unclaimed means its ready for a customer. claimed means its already in use by a customer. [destroyed] isn't a state you'll see in dynamodb, but it means that everything has been torn down.

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

### TODOs
1. JITProvisioner needs to return proper errors
1. Create and use a different kms key for installers
1. Sane scale levels for prod
1. Allow for parallel spinup of sandbox instances (preprovisioner)
1. https://redis.io/commands/flushdb/ during the teardown process
1. name state machines something random and track the new name in dynamodb
