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
